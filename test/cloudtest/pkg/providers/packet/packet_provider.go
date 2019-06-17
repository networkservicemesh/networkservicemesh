package shell

import (
	"bufio"
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/config"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/execmanager"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/k8s"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/providers"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/shell"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/utils"
	"github.com/packethost/packngo"
	"github.com/sirupsen/logrus"
	"log"
	"math/rand"
	"os"
	"path"
	"strings"
	"sync"
	"time"
)

const (
	InstallScript = "install" //#1
	SetupScript   = "setup"   //#2
	StartScript   = "start"   //#4
	ConfigScript  = "config"  //#5
	PrepareScript = "prepare" //#6
	StopScript    = "stop"    // #7
)

type packetProvider struct {
	root    string
	indexes map[string]int
	sync.Mutex
	clusters    []packetInstance
	installDone map[string]bool
}

type packetInstance struct {
	manager execmanager.ExecutionManager

	root               string
	id                 string
	config             *config.ClusterProviderConfig
	started            bool
	startFailed        int
	configScript       string
	installScript      []string
	setupScript      []string
	startScript        []string
	prepareScript      []string
	stopScript         []string
	zoneSelectorScript string
	provider           *packetProvider
	factory            k8s.ValidationFactory
	validator          k8s.KubernetesValidator
	configLocation     string

	shellInterface shell.ShellInterface
	params         providers.InstanceOptions
	client         *packngo.Client
	projectId      string
	packetAuthKey  string
	project        *packngo.Project
	genId          string
	devices        map[string]*packngo.Device
	keyId          string
	sshKey         *packngo.SSHKey
}

func (pi *packetInstance) GetId() string {
	return pi.id
}

func (pi *packetInstance) CheckIsAlive() error {
	if pi.started {
		return pi.validator.Validate()
	}
	return fmt.Errorf("Cluster is not running")
}

func (pi *packetInstance) IsRunning() bool {
	return pi.started
}

func (pi *packetInstance) GetClusterConfig() (string, error) {
	if pi.started {
		return pi.configLocation, nil
	}
	return "", fmt.Errorf("Cluster is not started yet...")
}

func (pi *packetInstance) Start(timeout time.Duration) error {
	logrus.Infof("Starting cluster %s-%s", pi.config.Name, pi.id)

	context, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Set seed
	rand.Seed(time.Now().UnixNano())

	utils.ClearFolder(pi.root, true)

	pi.genId = uuid.New().String()[:30]
	// Process and prepare enviorment variables
	err := pi.shellInterface.ProcessEnvironment(map[string]string{
		"cluster-uuid": pi.genId,
	})
	if err != nil {
		logrus.Errorf("Error during preocessing enviornment variables %v", err)
		return err
	}

	// Do prepare
	if !pi.params.NoInstall {
		if err := pi.doInstall(context); err != nil {
			return err
		}
	}

	// Run start script
	if err := pi.shellInterface.RunCmd(context, "setup", pi.setupScript, nil); err != nil {
		return err
	}


	keyFile := pi.config.Packet.SshKey
	if !utils.FileExists(keyFile) {
		// Relative file
		keyFile = path.Join(pi.root, keyFile)
		if !utils.FileExists(keyFile) {
			err := fmt.Errorf("Failed to locate generated key file, please specify init script to generate it.")
			logrus.Errorf(err.Error())
			return err
		}
	}

	pi.client, err = packngo.NewClient()
	if err != nil {
		logrus.Errorf("Failed to create Packet REST interface")
		return err

	}

	err = pi.updateProject()
	if err != nil {
		return err
	}

	// Check and add key if it is not yet added.

	pi.keyId = "dev-ci-cloud-" + pi.genId

	keyFileContent, err := utils.ReadFile(keyFile)
	if err != nil {
		logrus.Errorf("Failed to read file %v %v", keyFile, err)
		return err
	}

	keyRequest := &packngo.SSHKeyCreateRequest{
		ProjectID: pi.project.ID,
		Label:     pi.keyId,
		Key:       strings.Join(keyFileContent, "\n"),
	}
	sshKey, _, _ := pi.client.SSHKeys.Create(keyRequest)

	sshKeys, response, err := pi.client.SSHKeys.List()

	keyIds := []string{}
	for _, k := range sshKeys {
		if k.Label == pi.keyId {
			sshKey = &packngo.SSHKey{
				ID:          k.ID,
				Label:       k.Label,
				URL:         k.URL,
				User:        k.User,
				Key:         k.Key,
				FingerPrint: k.FingerPrint,
				Created:     k.Created,
				Updated:     k.Updated,
			}
		}
		keyIds = append(keyIds, k.ID)
	}

	if sshKey == nil && err != nil {
		logrus.Errorf("Failed to create ssh key %v", err)
		return err
	}

	pi.sshKey = sshKey
	pi.manager.AddLog(pi.id, "create-sshkey", fmt.Sprintf("%v\n%v\n%v", sshKey, response, err))

	facilitiesList, err := pi.findFacilities()

	for _, devCfg := range pi.config.Packet.Devices {
		devReq := &packngo.DeviceCreateRequest{
			//Facility:
			Plan:           devCfg.Plan,
			Facility:       facilitiesList,
			Hostname:       devCfg.Name + "-" + pi.genId,
			BillingCycle:   devCfg.BillingCycle,
			OS:             devCfg.OperatingSystem,
			ProjectID:      pi.projectId,
			ProjectSSHKeys: keyIds,
		}
		device, response, err := pi.client.Devices.Create(devReq)
		pi.manager.AddLog(pi.id, fmt.Sprintf("create-device-%s", devCfg.Name), fmt.Sprintf("%v", response))
		if err != nil {
			return err
		}

		pi.devices[devCfg.Name] = device
	}

	// All devices are created so we need to wait for them to get alive.

	for {
		alive := map[string]*packngo.Device{}
		for key, d := range pi.devices {
			dUpd, _, err := pi.client.Devices.Get(d.ID, &packngo.GetOptions{})
			if err != nil {
				logrus.Errorf("Error %v", err)
			} else {
				if dUpd.State == "active" {
					alive[key] = dUpd
				}
			}
		}
		if len(alive) == len(pi.devices) {
			pi.devices = alive
			break
		}
		select {
		case <-time.After(100 * time.Millisecond):
			continue
		case <-context.Done():
			logrus.Errorf("Timeout waiting for devices...")
			return fmt.Errorf("Timeout %v", context.Err())
		}
	}

	// We need to add arguments

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

	printableEnv := pi.shellInterface.PrintEnv(pi.shellInterface.GetProcessedEnv())
	pi.manager.AddLog(pi.id, "environment", printableEnv)

	// Run start script
	if err := pi.shellInterface.RunCmd(context, "start", pi.startScript, nil); err != nil {
		return err
	}

	if pi.configLocation == "" {
		pi.configLocation = pi.shellInterface.GetConfigLocation()
	}

	if pi.configLocation == "" {
		output, err := utils.ExecRead(context, strings.Split(pi.configScript, " "))
		if err != nil {
			msg := fmt.Sprintf("Failed to retrieve configuration location %v", err)
			logrus.Errorf(msg)
			return err
		}
		pi.configLocation = output[0]
	}

	pi.validator, err = pi.factory.CreateValidator(pi.config, pi.configLocation)
	if err != nil {
		msg := fmt.Sprintf("Failed to start validator %v", err)
		logrus.Errorf(msg)
		return err
	}
	// Run prepare script
	if err := pi.shellInterface.RunCmd(context, "prepare", pi.prepareScript, []string{"KUBECONFIG=" + pi.configLocation}); err != nil {
		return err
	}

	pi.started = true

	return nil
}

func (pi *packetInstance) findFacilities() ([]string, error) {
	facilities, response, err := pi.client.Facilities.List(&packngo.ListOptions{})

	pi.manager.AddLog(pi.id, "list-facilities", response.String())

	if err != nil {
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
	logrus.Infof("List of facilities: %v %v", facilities, response)

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
		selected := facilitiesList[ind]

		facilitiesList[ind] = facilitiesList[0]
		facilitiesList[0] = selected
	}

	return facilitiesList, nil
}

func (pi *packetInstance) Destroy(timeout time.Duration) error {
	logrus.Infof("Destroying cluster  %s", pi.id)
	response, err := pi.client.SSHKeys.Delete(pi.sshKey.ID)
	pi.manager.AddLog(pi.id, "delete-sshkey", fmt.Sprintf("%v\n%v\n%v", pi.sshKey, response, err))
	for key, device := range pi.devices {
		response, err := pi.client.Devices.Delete(device.ID)
		pi.manager.AddLog(pi.id, fmt.Sprintf("delete-device-%s", key), fmt.Sprintf("%v\n%v", response, err))
	}
	return nil
}

func (pi *packetInstance) GetRoot() string {
	return pi.root
}

func (pi *packetInstance) doDestroy(writer *bufio.Writer, timeout time.Duration, err error) {
	_, _ = writer.WriteString(fmt.Sprintf("Error during k8s API initialisation %v", err))
	_, _ = writer.WriteString(fmt.Sprintf("Trying to destroy cluster"))
	// In case we failed to start and create cluster utils.
	err2 := pi.Destroy(timeout)
	if err2 != nil {
		_, _ = writer.WriteString(fmt.Sprintf("Error during destroy of cluster %v", err2))
	}
}

func (pi *packetInstance) doInstall(context context.Context) error {
	pi.provider.Lock()
	defer pi.provider.Unlock()
	if pi.installScript != nil && !pi.provider.installDone[pi.config.Name] {
		pi.provider.installDone[pi.config.Name] = true
		return pi.shellInterface.RunCmd(context, "install", pi.installScript, nil)
	}
	return nil
}

func (pi *packetInstance) updateProject() error {
	ps, response, err := pi.client.Projects.List(nil)

	pi.manager.AddLog(pi.id, "list-projects", fmt.Sprintf("%v\n%v\n%v", ps, response, err))

	if err != nil {
		logrus.Errorf("Failed to list Packet projects")
	}

	for _, p := range ps {
		log.Println(p.ID, p.Name)
		if p.ID == pi.projectId {
			pp := p
			pi.project = &pp
			break
		}
	}

	if pi.project == nil {
		err := fmt.Errorf("Specified project are not found on Packet %v", pi.projectId)
		logrus.Errorf(err.Error())
		return err
	}
	return nil
}

func (p *packetProvider) getProviderId(provider string) string {
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
	id := fmt.Sprintf("%s-%s", config.Name, p.getProviderId(config.Name))

	root := path.Join(p.root, id)

	clusterInstance := &packetInstance{
		manager:            manager,
		provider:           p,
		root:               root,
		id:                 id,
		config:             config,
		configScript:       config.Scripts[ConfigScript],
		installScript:      utils.ParseScript(config.Scripts[InstallScript]),
		setupScript:        utils.ParseScript(config.Scripts[SetupScript]),
		startScript:        utils.ParseScript(config.Scripts[StartScript]),
		prepareScript:      utils.ParseScript(config.Scripts[PrepareScript]),
		stopScript:         utils.ParseScript(config.Scripts[StopScript]),
		factory:            factory,
		shellInterface:     shell.NewShellInterface(manager, id, root, config, instanceOptions),
		params:             instanceOptions,
		projectId:          os.Getenv("PACKET_PROJECT_ID"),
		packetAuthKey:      os.Getenv("PACKET_AUTH_TOKEN"),
		devices:            map[string]*packngo.Device{},
	}

	return clusterInstance, nil
}

func init() {
	logrus.Infof("Adding packet as supported providers...")
	providers.ClusterProviderFactories["packet"] = NewPacketClusterProvider
}

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
		return fmt.Errorf("Packet configuration element should be specified...")
	}

	if len(config.Packet.Facilities) == 0 {
		return fmt.Errorf("Packet configuration facilities should be specified...")
	}

	if len(config.Packet.Devices) == 0 {
		return fmt.Errorf("Packet configuration devices should be specified...")
	}

	if _, ok := config.Scripts[ConfigScript]; !ok {
		hasKubeConfig := false
		for _, e := range config.Env {
			if strings.HasPrefix(e, "KUBECONFIG=") {
				hasKubeConfig = true
				break
			}
		}
		if !hasKubeConfig {
			return fmt.Errorf("Invalid config location")
		}
	}
	if _, ok := config.Scripts[StartScript]; !ok {
		return fmt.Errorf("Invalid start script")
	}

	for _, envVar := range config.EnvCheck {
		envValue := os.Getenv(envVar)
		if envValue == "" {
			return fmt.Errorf("Environment variable are not specified %s Required variables: %v", envValue, config.EnvCheck)
		}
	}

	envValue := os.Getenv("PACKET_AUTH_TOKEN")
	if envValue == "" {
		return fmt.Errorf("Environment variable are not specified PACKET_AUTH_TOKEN")
	}

	envValue = os.Getenv("PACKET_PROJECT_ID")
	if envValue == "" {
		return fmt.Errorf("Environment variable are not specified PACKET_AUTH_TOKEN")
	}

	return nil
}
