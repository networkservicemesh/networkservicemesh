package packet

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/packethost/packngo"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/config"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/execmanager"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/k8s"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/providers"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/shell"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/utils"
)

const (
	installScript   = "install" //#1
	setupScript     = "setup"   //#2
	startScript     = "start"   //#4
	configScript    = "config"  //#5
	prepareScript   = "prepare" //#6
	stopScript      = "stop"    // #7
	packetProjectID = "PACKET_PROJECT_ID"
)

type packetProvider struct {
	root    string
	indexes map[string]int
	sync.Mutex
	clusters    []packetInstance
	installDone map[string]bool
}

type packetInstance struct {
	installScript  []string
	setupScript    []string
	startScript    []string
	prepareScript  []string
	stopScript     []string
	manager        execmanager.ExecutionManager
	root           string
	id             string
	configScript   string
	factory        k8s.ValidationFactory
	validator      k8s.KubernetesValidator
	configLocation string
	shellInterface shell.Manager
	projectID      string
	packetAuthKey  string
	keyID          string
	config         *config.ClusterProviderConfig
	provider       *packetProvider
	client         *packngo.Client
	project        *packngo.Project
	devices        map[string]*packngo.Device
	sshKey         *packngo.SSHKey
	params         providers.InstanceOptions
	started        bool
	keyIds         []string
	facilitiesList []string
}

func (pi *packetInstance) GetID() string {
	return pi.id
}

func (pi *packetInstance) CheckIsAlive() error {
	if pi.started {
		return pi.validator.Validate()
	}
	return errors.New("cluster is not running")
}

func (pi *packetInstance) IsRunning() bool {
	return pi.started
}

func (pi *packetInstance) GetClusterConfig() (string, error) {
	if pi.started {
		return pi.configLocation, nil
	}
	return "", errors.New("cluster is not started yet")
}

func (pi *packetInstance) Start(timeout time.Duration) (string, error) {
	logrus.Infof("Starting cluster %s-%s", pi.config.Name, pi.id)
	var err error
	fileName := ""
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Set seed
	rand.Seed(time.Now().UnixNano())

	utils.ClearFolder(pi.root, true)

	// Process and prepare environment variables
	if err = pi.shellInterface.ProcessEnvironment(
		pi.id, pi.config.Name, pi.root, pi.config.Env, nil); err != nil {
		logrus.Errorf("error during processing environment variables %v", err)
		return "", err
	}

	// Do prepare
	if !pi.params.NoInstall {
		if fileName, err = pi.doInstall(ctx); err != nil {
			return fileName, err
		}
	}

	// Run start script
	if fileName, err = pi.shellInterface.RunCmd(ctx, "setup", pi.setupScript, nil); err != nil {
		return fileName, err
	}

	keyFile := pi.config.Packet.SshKey
	if !utils.FileExists(keyFile) {
		// Relative file
		keyFile = path.Join(pi.root, keyFile)
		if !utils.FileExists(keyFile) {
			err = errors.New("failed to locate generated key file, please specify init script to generate it")
			logrus.Errorf(err.Error())
			return "", err
		}
	}

	if pi.client, err = packngo.NewClient(); err != nil {
		logrus.Errorf("failed to create Packet REST interface")
		return "", err
	}

	if err = pi.updateProject(); err != nil {
		return "", err
	}

	// Check and add key if it is not yet added.

	if pi.keyIds, err = pi.createKey(keyFile); err != nil {
		return "", err
	}

	if pi.facilitiesList, err = pi.findFacilities(); err != nil {
		return "", err
	}
	for _, devCfg := range pi.config.Packet.Devices {
		var device *packngo.Device
		if device, err = pi.createDevice(devCfg); err != nil {
			return "", err
		}
		pi.devices[devCfg.Name] = device
	}

	// All devices are created so we need to wait for them to get alive.
	if err = pi.waitDevicesStartup(ctx); err != nil {
		return "", err
	}
	// We need to add arguments

	pi.addDeviceContextArguments()

	printableEnv := pi.shellInterface.PrintEnv(pi.shellInterface.GetProcessedEnv())
	pi.manager.AddLog(pi.id, "environment", printableEnv)

	// Run start script
	if fileName, err = pi.shellInterface.RunCmd(ctx, "start", pi.startScript, nil); err != nil {
		return fileName, err
	}

	if err = pi.updateKUBEConfig(ctx); err != nil {
		return "", err
	}

	if pi.validator, err = pi.factory.CreateValidator(pi.config, pi.configLocation); err != nil {
		msg := fmt.Sprintf("Failed to start validator %v", err)
		logrus.Errorf(msg)
		return "", err
	}
	// Run prepare script
	if fileName, err = pi.shellInterface.RunCmd(ctx, "prepare", pi.prepareScript, []string{"KUBECONFIG=" + pi.configLocation}); err != nil {
		return fileName, err
	}

	// Wait a bit to be sure clusters are up and running.
	st := time.Now()
	err = pi.validator.WaitValid(ctx)
	if err != nil {
		logrus.Errorf("Failed to wait for required number of nodes: %v", err)
		return fileName, err
	}
	logrus.Infof("Waiting for desired number of nodes complete %s-%s %v", pi.config.Name, pi.id, time.Since(st))

	pi.started = true
	logrus.Infof("Starting are up and running %s-%s", pi.config.Name, pi.id)
	return "", nil
}

func (pi *packetInstance) updateKUBEConfig(context context.Context) error {
	if pi.configLocation == "" {
		pi.configLocation = pi.shellInterface.GetConfigLocation()
	}
	if pi.configLocation == "" {
		output, err := utils.ExecRead(context, "", strings.Split(pi.configScript, " "))
		if err != nil {
			err = errors.Wrap(err, "failed to retrieve configuration location")
			logrus.Errorf(err.Error())
		}
		pi.configLocation = output[0]
	}
	return nil
}

func (pi *packetInstance) addDeviceContextArguments() {
	for key, dev := range pi.devices {
		for _, n := range dev.Network {
			pub := "pub"
			if !n.Public {
				pub = "private"
			}
			pi.shellInterface.AddExtraArgs(fmt.Sprintf("device.%v.%v.%v.%v", key, pub, "ip", n.AddressFamily), n.Address)
			pi.shellInterface.AddExtraArgs(fmt.Sprintf("device.%v.%v.%v.%v", key, pub, "gw", n.AddressFamily), n.Gateway)
			pi.shellInterface.AddExtraArgs(fmt.Sprintf("device.%v.%v.%v.%v", key, pub, "net", n.AddressFamily), n.Network)
		}
	}
}

func (pi *packetInstance) waitDevicesStartup(context context.Context) error {
	_, fileID, err := pi.manager.OpenFile(pi.id, "wait-nodes")
	if err != nil {
		return err
	}

	writer := bufio.NewWriter(fileID)
	defer func() { _ = fileID.Close() }()
	for {
		alive := map[string]*packngo.Device{}
		for key, d := range pi.devices {
			var updatedDevice *packngo.Device
			updatedDevice, _, err := pi.client.Devices.Get(d.ID, &packngo.GetOptions{})
			if err != nil {
				logrus.Errorf("%v-%v Error accessing device Error: %v", pi.id, d.ID, err)
				continue
			} else if updatedDevice.State == "active" {
				alive[key] = updatedDevice
			}
			msg := fmt.Sprintf("Checking status %v %v %v", key, d.ID, updatedDevice.State)
			_, _ = writer.WriteString(msg)
			_ = writer.Flush()
			logrus.Infof("%v-Checking status %v", pi.id, updatedDevice.State)
		}
		if len(alive) == len(pi.devices) {
			pi.devices = alive
			break
		}
		select {
		case <-time.After(10 * time.Second):
			continue
		case <-context.Done():
			_, _ = writer.WriteString(fmt.Sprintf("Timeout"))
			return errors.Wrap(context.Err(), "timeout")
		}
	}
	_, _ = writer.WriteString(fmt.Sprintf("All devices online"))
	_ = writer.Flush()
	return nil
}

func (pi *packetInstance) createDevice(devCfg *config.DeviceConfig) (*packngo.Device, error) {
	finalEnv := pi.shellInterface.GetProcessedEnv()

	environment := map[string]string{}
	for _, k := range finalEnv {
		key, value, err := utils.ParseVariable(k)
		if err != nil {
			return nil, err
		}
		environment[key] = value
	}
	var hostName string
	var err error
	if hostName, err = utils.SubstituteVariable(devCfg.HostName, environment, pi.shellInterface.GetArguments()); err != nil {
		return nil, err
	}

	devReq := &packngo.DeviceCreateRequest{
		Plan:           devCfg.Plan,
		Facility:       pi.facilitiesList,
		Hostname:       hostName,
		BillingCycle:   devCfg.BillingCycle,
		OS:             devCfg.OperatingSystem,
		ProjectID:      pi.projectID,
		ProjectSSHKeys: pi.keyIds,
	}
	var device *packngo.Device
	var response *packngo.Response
	for {
		device, response, err = pi.client.Devices.Create(devReq)
		msg := fmt.Sprintf("HostName=%v\n%v - %v", hostName, response, err)
		logrus.Infof(fmt.Sprintf("%s-%v", pi.id, msg))
		pi.manager.AddLog(pi.id, fmt.Sprintf("create-device-%s", devCfg.Name), msg)
		if err == nil || err != nil && !strings.Contains(err.Error(), "has no provisionable") || len(devReq.Facility) <= 1 {
			break
		}

		devReq.Facility = devReq.Facility[1:]
	}
	return device, err
}

func (pi *packetInstance) findFacilities() ([]string, error) {
	facilities, response, err := pi.client.Facilities.List(&packngo.ListOptions{})

	out := strings.Builder{}
	_, _ = out.WriteString(fmt.Sprintf("%v\n%v\n", response.String(), err))

	if err != nil {
		pi.manager.AddLog(pi.id, "list-facilities", out.String())
		return nil, err
	}

	facilitiesList := []string{}
	for _, f := range facilities {
		facilityReqs := map[string]string{}
		for _, ff := range f.Features {
			facilityReqs[ff] = ff
		}

		found := true
		for _, ff := range pi.config.Packet.Facilities {
			if _, ok := facilityReqs[ff]; !ok {
				found = false
				break
			}
		}
		if found {
			facilitiesList = append(facilitiesList, f.Code)
		}
	}

	// Randomize facilities.
	ind := -1

	if pi.config.Packet.PreferredFacility != "" {
		for i, f := range facilitiesList {
			if f == pi.config.Packet.PreferredFacility {
				ind = i
				break
			}
		}
	}

	if ind != -1 {
		facilitiesList[ind], facilitiesList[0] = facilitiesList[0], facilitiesList[ind]
	}

	msg := fmt.Sprintf("List of facilities: %v %v", facilities, response)
	//logrus.Infof(msg)
	_, _ = out.WriteString(msg)
	pi.manager.AddLog(pi.id, "list-facilities", out.String())

	return facilitiesList, nil
}

func (pi *packetInstance) Destroy(timeout time.Duration) error {
	logrus.Infof("Destroying cluster  %s", pi.id)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if pi.client != nil {
		if pi.sshKey != nil {
			response, err := pi.client.SSHKeys.Delete(pi.sshKey.ID)
			pi.manager.AddLog(pi.id, "delete-sshkey", fmt.Sprintf("%v\n%v\n%v", pi.sshKey, response, err))
		}

		_, logFile, err := pi.manager.OpenFile(pi.id, "destroy-cluster")
		defer func() { _ = logFile.Close() }()
		if err != nil {
			return err
		}
		_, _ = logFile.WriteString(fmt.Sprintf("Starting Delete of cluster %v", pi.id))
		iteration := 0
		for {
			alive := map[string]*packngo.Device{}
			for key, device := range pi.devices {
				var updatedDevice *packngo.Device
				updatedDevice, _, err := pi.client.Devices.Get(device.ID, &packngo.GetOptions{})
				if err != nil {
					if iteration == 0 {
						msg := fmt.Sprintf("%v-%v Error accessing device Error: %v", pi.id, device.ID, err)
						logrus.Error(msg)
						_, _ = logFile.WriteString(msg)
					} // else, if not first iteration and there is no device, just continue.
					continue
				}
				if updatedDevice.State != "provisioning" && updatedDevice.State != "queued" {
					response, err := pi.client.Devices.Delete(device.ID)
					if err != nil {
						_, _ = logFile.WriteString(fmt.Sprintf("delete-device-error-%s => %v\n%v ", key, response, err))
						logrus.Errorf("%v Failed to delete device %v", pi.id, device.ID)
					} else {
						_, _ = logFile.WriteString(fmt.Sprintf("delete-device-success-%s => %v\n%v ", key, response, err))
						logrus.Infof("%v Packet delete device send ok %v", pi.id, device.ID)
					}
				}
				// Put as alive or some different state
				alive[key] = updatedDevice

				msg := fmt.Sprintf("Device status %v %v %v", key, device.ID, updatedDevice.State)
				_, _ = logFile.WriteString(msg)
				logrus.Infof("%v-%v", pi.id, msg)
			}
			iteration++
			if len(alive) == 0 {
				break
			}
			select {
			case <-time.After(10 * time.Second):
				continue
			case <-ctx.Done():
				msg := fmt.Sprintf("Timeout for destroying cluster devices %v %v", pi.devices, ctx.Err())
				_, _ = logFile.WriteString(msg)
				return errors.Errorf("err: %v", msg)
			}
		}
		msg := fmt.Sprintf("Devices destroy complete %v", pi.devices)
		_, _ = logFile.WriteString(msg)
		logrus.Infof("Destroy Complete: %v", pi.id)
	}
	return nil
}

func (pi *packetInstance) GetRoot() string {
	return pi.root
}

func (pi *packetInstance) doDestroy(writer io.StringWriter, timeout time.Duration, err error) {
	_, _ = writer.WriteString(fmt.Sprintf("Error during k8s API initialisation %v", err))
	_, _ = writer.WriteString(fmt.Sprintf("Trying to destroy cluster"))
	// In case we failed to start and create cluster utils.
	err2 := pi.Destroy(timeout)
	if err2 != nil {
		_, _ = writer.WriteString(fmt.Sprintf("Error during destroy of cluster %v", err2))
	}
}

func (pi *packetInstance) doInstall(context context.Context) (string, error) {
	pi.provider.Lock()
	defer pi.provider.Unlock()
	if pi.installScript != nil && !pi.provider.installDone[pi.config.Name] {
		pi.provider.installDone[pi.config.Name] = true
		return pi.shellInterface.RunCmd(context, "install", pi.installScript, nil)
	}
	return "", nil
}

func (pi *packetInstance) updateProject() error {
	ps, response, err := pi.client.Projects.List(nil)

	out := strings.Builder{}
	_, _ = out.WriteString(fmt.Sprintf("%v\n%v\n", response, err))

	if err != nil {
		logrus.Errorf("Failed to list Packet projects")
	}

	for i := 0; i < len(ps); i++ {
		p := &ps[i]
		_, _ = out.WriteString(fmt.Sprintf("Project: %v\n %v", p.Name, p))
		if p.ID == pi.projectID {
			pp := ps[i]
			pi.project = &pp
		}
	}

	pi.manager.AddLog(pi.id, "list-projects", out.String())

	if pi.project == nil {
		err := errors.Errorf("%s - specified project are not found on Packet %v", pi.id, pi.projectID)
		logrus.Errorf(err.Error())
		return err
	}
	return nil
}

func (pi *packetInstance) createKey(keyFile string) ([]string, error) {
	today := time.Now()
	genID := fmt.Sprintf("%d-%d-%d-%s", today.Year(), today.Month(), today.Day(), utils.NewRandomStr(10))
	pi.keyID = "dev-ci-cloud-" + genID

	out := strings.Builder{}
	keyFileContent, err := utils.ReadFile(keyFile)
	if err != nil {
		_, _ = out.WriteString(fmt.Sprintf("Failed to read key file %s", keyFile))
		pi.manager.AddLog(pi.id, "create-key", out.String())
		logrus.Errorf("Failed to read file %v %v", keyFile, err)
		return nil, err
	}

	_, _ = out.WriteString(fmt.Sprintf("Key file %s readed ok", keyFile))

	keyRequest := &packngo.SSHKeyCreateRequest{
		ProjectID: pi.project.ID,
		Label:     pi.keyID,
		Key:       strings.Join(keyFileContent, "\n"),
	}
	sshKey, response, err := pi.client.SSHKeys.Create(keyRequest)

	responseMsg := ""
	if response != nil {
		responseMsg = response.String()
	}
	createMsg := fmt.Sprintf("Create key %v %v %v", sshKey, responseMsg, err)
	_, _ = out.WriteString(createMsg)

	keyIds := []string{}
	if sshKey == nil {
		// try to find key.
		sshKey, keyIds = pi.findKeys(&out)
	} else {
		logrus.Infof("%s-Create key %v (%v)", pi.id, sshKey.ID, sshKey.Key)
		keyIds = append(keyIds, sshKey.ID)
	}
	pi.sshKey = sshKey
	pi.manager.AddLog(pi.id, "create-sshkey", fmt.Sprintf("%v\n%v\n%v\n %s", sshKey, response, err, out.String()))

	if sshKey == nil {
		_, _ = out.WriteString(fmt.Sprintf("Failed to create ssh key %v %v", sshKey, err))
		pi.manager.AddLog(pi.id, "create-key", out.String())
		logrus.Errorf("Failed to create ssh key %v", err)
		return nil, err
	}
	return keyIds, nil
}

func (pi *packetInstance) findKeys(out io.StringWriter) (*packngo.SSHKey, []string) {
	sshKeys, response, err := pi.client.SSHKeys.List()
	if err != nil {
		_, _ = out.WriteString(fmt.Sprintf("List keys error %v %v\n", response, err))
	}
	var keyIds []string
	var sshKey *packngo.SSHKey
	for k := 0; k < len(sshKeys); k++ {
		kk := &sshKeys[k]
		if kk.Label == pi.keyID {
			sshKey = &packngo.SSHKey{
				ID:          kk.ID,
				Label:       kk.Label,
				URL:         kk.URL,
				User:        kk.User,
				Key:         kk.Key,
				FingerPrint: kk.FingerPrint,
				Created:     kk.Created,
				Updated:     kk.Updated,
			}
		}
		_, _ = out.WriteString(fmt.Sprintf("Added key key %v\n", kk))
		keyIds = append(keyIds, kk.ID)
	}
	return sshKey, keyIds
}

func (p *packetProvider) getProviderID(provider string) string {
	val, ok := p.indexes[provider]
	if ok {
		val++
	} else {
		val = 1
	}
	p.indexes[provider] = val
	return fmt.Sprintf("%d", val)
}

func (p *packetProvider) CreateCluster(config *config.ClusterProviderConfig, factory k8s.ValidationFactory,
	manager execmanager.ExecutionManager,
	instanceOptions providers.InstanceOptions) (providers.ClusterInstance, error) {
	err := p.ValidateConfig(config)
	if err != nil {
		return nil, err
	}
	p.Lock()
	defer p.Unlock()
	id := fmt.Sprintf("%s-%s", config.Name, p.getProviderID(config.Name))

	root := path.Join(p.root, id)

	clusterInstance := &packetInstance{
		manager:        manager,
		provider:       p,
		root:           root,
		id:             id,
		config:         config,
		configScript:   config.Scripts[configScript],
		installScript:  utils.ParseScript(config.Scripts[installScript]),
		setupScript:    utils.ParseScript(config.Scripts[setupScript]),
		startScript:    utils.ParseScript(config.Scripts[startScript]),
		prepareScript:  utils.ParseScript(config.Scripts[prepareScript]),
		stopScript:     utils.ParseScript(config.Scripts[stopScript]),
		factory:        factory,
		shellInterface: shell.NewManager(manager, id, config, instanceOptions),
		params:         instanceOptions,
		projectID:      os.Getenv(packetProjectID),
		packetAuthKey:  os.Getenv("PACKET_AUTH_TOKEN"),
		devices:        map[string]*packngo.Device{},
	}

	return clusterInstance, nil
}

// NewPacketClusterProvider - create new packet provider.
func NewPacketClusterProvider(root string) providers.ClusterProvider {
	utils.ClearFolder(root, true)
	return &packetProvider{
		root:        root,
		clusters:    []packetInstance{},
		indexes:     map[string]int{},
		installDone: map[string]bool{},
	}
}

func (p *packetProvider) ValidateConfig(config *config.ClusterProviderConfig) error {
	if config.Packet == nil {
		return errors.New("packet configuration element should be specified")
	}

	if len(config.Packet.Facilities) == 0 {
		return errors.New("packet configuration facilities should be specified")
	}

	if len(config.Packet.Devices) == 0 {
		return errors.New("packet configuration devices should be specified")
	}

	if _, ok := config.Scripts[configScript]; !ok {
		hasKubeConfig := false
		for _, e := range config.Env {
			if strings.HasPrefix(e, "KUBECONFIG=") {
				hasKubeConfig = true
				break
			}
		}
		if !hasKubeConfig {
			return errors.New("invalid config location")
		}
	}
	if _, ok := config.Scripts[startScript]; !ok {
		return errors.New("invalid start script")
	}

	for _, envVar := range config.EnvCheck {
		envValue := os.Getenv(envVar)
		if envValue == "" {
			return errors.Errorf("environment variable are not specified %s Required variables: %v", envValue, config.EnvCheck)
		}
	}

	envValue := os.Getenv("PACKET_AUTH_TOKEN")
	if envValue == "" {
		return errors.New("environment variable are not specified PACKET_AUTH_TOKEN")
	}

	envValue = os.Getenv("PACKET_PROJECT_ID")
	if envValue == "" {
		return errors.New("environment variable are not specified PACKET_AUTH_TOKEN")
	}

	return nil
}
