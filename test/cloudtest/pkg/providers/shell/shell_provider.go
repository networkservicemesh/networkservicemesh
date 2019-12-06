package shell

import (
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
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/config"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/execmanager"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/k8s"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/providers"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/shell"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/utils"
)

const (
	installScript = "install" //#1
	startScript   = "start"   //#2
	configScript  = "config"  //#3
	prepareScript = "prepare" //#4
	stopScript    = "stop"    // #5
	cleanupScript = "cleanup" // #6
	zoneSelector  = "zone-selector"
)

type shellProvider struct {
	root    string
	indexes map[string]int
	sync.Mutex
	clusters    []shellInstance
	installDone map[string]bool
}

type shellInstance struct {
	installScript      []string
	startScript        []string
	prepareScript      []string
	stopScript         []string
	manager            execmanager.ExecutionManager
	root               string
	id                 string
	configScript       string
	zoneSelectorScript []string
	factory            k8s.ValidationFactory
	validator          k8s.KubernetesValidator
	configLocation     string
	shellInterface     shell.Manager
	config             *config.ClusterProviderConfig
	provider           *shellProvider
	params             providers.InstanceOptions
	started            bool
}

func (si *shellInstance) GetID() string {
	return si.id
}

func (si *shellInstance) CheckIsAlive() error {
	if si.started {
		return si.validator.Validate()
	}
	return errors.New("cluster is not running")
}

func (si *shellInstance) IsRunning() bool {
	return si.started
}

func (si *shellInstance) GetClusterConfig() (string, error) {
	if si.started {
		return si.configLocation, nil
	}
	return "", errors.New("cluster is not started yet")
}

func (si *shellInstance) Start(timeout time.Duration) (string, error) {
	logrus.Infof("Starting cluster %s-%s", si.config.Name, si.id)

	context, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Set seed
	rand.Seed(time.Now().UnixNano())

	utils.ClearFolder(si.root, true)
	var err error
	fileName := ""

	// Do prepare
	if !si.params.NoInstall {
		if fileName, err = si.doInstall(context); err != nil {
			return fileName, err
		}
	}

	selectedZone, err := selectZone(context, si.shellInterface, si.zoneSelectorScript)
	if err != nil {
		return "", err
	}

	// Process and prepare environment variables
	err = si.shellInterface.ProcessEnvironment(
		si.id, si.config.Name, si.root, si.config.Env,
		map[string]string{
			"zone-selector": selectedZone,
		})
	if err != nil {
		return "", err
	}

	printableEnv := si.shellInterface.PrintEnv(si.shellInterface.GetProcessedEnv())
	si.manager.AddLog(si.id, "environment", printableEnv)

	// Run start script
	if fileName, err = si.shellInterface.RunCmd(context, "start", si.startScript, nil); err != nil {
		return fileName, err
	}

	if si.configLocation == "" {
		si.configLocation = si.shellInterface.GetConfigLocation()
	}

	if si.configLocation == "" {
		var output []string
		output, err = utils.ExecRead(context, "", strings.Split(si.configScript, " "))
		if err != nil {
			msg := fmt.Sprintf("Failed to retrieve configuration location %v", err)
			logrus.Errorf(msg)
			return "", err
		}
		si.configLocation = output[0]
	}
	si.validator, err = si.factory.CreateValidator(si.config, si.configLocation)
	if err != nil {
		msg := fmt.Sprintf("Failed to start validator %v", err)
		logrus.Errorf(msg)
		return "", err
	}
	// Run prepare script
	if !si.params.NoPrepare {
		if fileName, err := si.shellInterface.RunCmd(context, "prepare", si.prepareScript, []string{"KUBECONFIG=" + si.configLocation}); err != nil {
			return fileName, err
		}
	}

	// Wait a bit to be sure clusters are up and running.
	st := time.Now()

	err = si.validator.WaitValid(context)
	if err != nil {
		logrus.Errorf("Failed to wait for required number of nodes: %v", err)
		return fileName, err
	}
	logrus.Infof("Waiting for desired number of nodes complete %s-%s %v", si.config.Name, si.id, time.Since(st))

	si.started = true

	return "", nil
}

func (si *shellInstance) Destroy(timeout time.Duration) error {
	logrus.Infof("Destroying cluster  %s", si.id)

	context, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	attempts := si.config.RetryCount
	for {
		_, err := si.shellInterface.RunCmd(context, fmt.Sprintf("destroy-%d", si.config.RetryCount-attempts), si.stopScript, nil)
		if err == nil || attempts == 0 {
			return err
		}
		attempts--
	}
}

func (si *shellInstance) GetRoot() string {
	return si.root
}

func (si *shellInstance) doDestroy(writer io.StringWriter, timeout time.Duration, err error) {
	_, _ = writer.WriteString(fmt.Sprintf("Error during k8s API initialisation %v", err))
	_, _ = writer.WriteString(fmt.Sprintf("Trying to destroy cluster"))
	// In case we failed to start and create cluster utils.
	err2 := si.Destroy(timeout)
	if err2 != nil {
		_, _ = writer.WriteString(fmt.Sprintf("Error during destroy of cluster %v", err2))
	}
}

func (si *shellInstance) doInstall(context context.Context) (string, error) {
	si.provider.Lock()
	defer si.provider.Unlock()
	if si.installScript != nil && !si.provider.installDone[si.config.Name] {
		si.provider.installDone[si.config.Name] = true
		return si.shellInterface.RunCmd(context, "install", si.installScript, nil)
	}
	return "", nil
}

func selectZone(ctx context.Context, shellInterface shell.Manager, zoneSelectorScript []string) (string, error) {
	selectedZone := ""

	if len(zoneSelectorScript) > 0 {
		zones, err := shellInterface.RunRead(ctx, zoneSelector, zoneSelectorScript, nil)
		if err != nil {
			logrus.Errorf("Failed to select zones...")
			return "", err
		}
		zonesList := strings.Split(zones, "\n")
		if len(zonesList) == 0 {
			return "", errors.New("failed to retrieve a zone list")
		}

		selectedZone += zonesList[rand.Intn(len(zonesList))]
	}

	return selectedZone, nil
}

func (p *shellProvider) getProviderID(provider string) string {
	val, ok := p.indexes[provider]
	if ok {
		val++
	} else {
		val = 1
	}
	p.indexes[provider] = val
	return fmt.Sprintf("%d", val)
}

func (p *shellProvider) CreateCluster(config *config.ClusterProviderConfig, factory k8s.ValidationFactory,
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

	clusterInstance := &shellInstance{
		manager:            manager,
		provider:           p,
		root:               root,
		id:                 id,
		config:             config,
		configScript:       config.Scripts[configScript],
		installScript:      utils.ParseScript(config.Scripts[installScript]),
		startScript:        utils.ParseScript(config.Scripts[startScript]),
		prepareScript:      utils.ParseScript(config.Scripts[prepareScript]),
		stopScript:         utils.ParseScript(config.Scripts[stopScript]),
		zoneSelectorScript: utils.ParseScript(config.Scripts[zoneSelector]),
		factory:            factory,
		shellInterface:     shell.NewManager(manager, id, config, instanceOptions),
		params:             instanceOptions,
	}

	return clusterInstance, nil
}

// CleanupClusters - Cleaning up leaked clusters
func (p *shellProvider) CleanupClusters(ctx context.Context, config *config.ClusterProviderConfig,
	manager execmanager.ExecutionManager, instanceOptions providers.InstanceOptions) {
	clScript := utils.ParseScript(config.Scripts[cleanupScript])
	if clScript == nil {
		// Skip
		return
	}

	logrus.Infof("Starting cleaning up clusters for %s", config.Name)
	shellInterface := shell.NewManager(manager, fmt.Sprintf("%s-all", config.Name), config, instanceOptions)

	iScript := utils.ParseScript(config.Scripts[installScript])
	if iScript != nil {
		_, err := shellInterface.RunCmd(ctx, "install", iScript, config.Env)
		if err != nil {
			logrus.Warnf("Install command for cluster %s finished with error: %v", config.Name, err)
			return
		}
	}

	selectedZone, err := selectZone(ctx, shellInterface, utils.ParseScript(config.Scripts[zoneSelector]))
	if err != nil {
		logrus.Warnf("Select zone command for cluster %s finished with error: %v", config.Name, err)
		return
	}

	// Process and prepare environment variables
	err = shellInterface.ProcessEnvironment(
		"all", config.Name, p.root, config.Env,
		map[string]string{
			"zone-selector": selectedZone,
		})
	if err != nil {
		logrus.Warnf("Select zone command for cluster %s finished with error: %v", config.Name, err)
		return
	}

	_, err = shellInterface.RunCmd(ctx, "clScript", clScript, nil)
	if err != nil {
		logrus.Warnf("Cleanup command for cluster %s finished with error: %v", config.Name, err)
	}
}

// NewShellClusterProvider - Creates new shell provider
func NewShellClusterProvider(root string) providers.ClusterProvider {
	utils.ClearFolder(root, true)
	return &shellProvider{
		root:        root,
		clusters:    []shellInstance{},
		indexes:     map[string]int{},
		installDone: map[string]bool{},
	}
}

func (p *shellProvider) ValidateConfig(config *config.ClusterProviderConfig) error {
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
	if _, ok := config.Scripts[stopScript]; !ok {
		return errors.New("invalid shutdown script location")
	}

	for _, envVar := range config.EnvCheck {
		envValue := os.Getenv(envVar)
		if envValue == "" {
			return errors.Errorf("environment variable are not specified %s Required variables: %v", envValue, config.EnvCheck)
		}
	}

	return nil
}
