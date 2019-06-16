package shell

import (
	"bufio"
	"context"
	"fmt"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/config"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/execmanager"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/k8s"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/providers"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/shell"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/utils"
	"github.com/sirupsen/logrus"
	"math/rand"
	"os"
	"path"
	"strings"
	"sync"
	"time"
)

const (
	InstallScript = "install" //#1
	StartScript   = "start"   //#2
	ConfigScript  = "config"  //#3
	PrepareScript = "prepare" //#4
	StopScript    = "stop"    // #5
	ZoneSelector  = "zone-selector"
)

type shellProvider struct {
	root    string
	indexes map[string]int
	sync.Mutex
	clusters    []shellInstance
	installDone map[string]bool
}

type shellInstance struct {
	manager execmanager.ExecutionManager

	root               string
	id                 string
	config             *config.ClusterProviderConfig
	started            bool
	startFailed        int
	configScript       string
	installScript      []string
	startScript        []string
	prepareScript      []string
	stopScript         []string
	zoneSelectorScript string
	provider           *shellProvider
	factory            k8s.ValidationFactory
	validator          k8s.KubernetesValidator
	configLocation     string

	shellInterface shell.ShellInterface
	params         providers.InstanceOptions
}

func (si *shellInstance) GetId() string {
	return si.id
}

func (si *shellInstance) CheckIsAlive() error {
	if si.started {
		return si.validator.Validate()
	}
	return fmt.Errorf("Cluster is not running")
}

func (si *shellInstance) IsRunning() bool {
	return si.started
}

func (si *shellInstance) GetClusterConfig() (string, error) {
	if si.started {
		return si.configLocation, nil
	}
	return "", fmt.Errorf("Cluster is not started yet...")
}

func (si *shellInstance) Start(timeout time.Duration) error {
	logrus.Infof("Starting cluster %s-%s", si.config.Name, si.id)

	context, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Set seed
	rand.Seed(time.Now().UnixNano())

	utils.ClearFolder(si.root, true)

	selectedZone := ""

	if si.zoneSelectorScript != "" {
		zones, err := utils.ExecRead(context, strings.Split(si.zoneSelectorScript, " "))
		if err != nil {
			logrus.Errorf("Failed to select zones...")
			return err
		}
		selectedZone += zones[rand.Intn(len(zones)-1)]
	}

	// Process and prepare enviorment variables
	si.shellInterface.ProcessEnvironment(map[string]string{
		"zone-selector": selectedZone,
	})

	// Do prepare
	if !si.params.NoInstall {
		if err := si.doInstall(context); err != nil {
			return err
		}
	}

	printableEnv := si.shellInterface.PrintEnv(si.shellInterface.GetProcessedEnv())
	si.manager.AddLog(si.id, "environment", printableEnv)

	// Run start script
	if err := si.shellInterface.RunCmd(context, "start", si.startScript, nil); err != nil {
		return err
	}

	if si.configLocation == "" {
		si.configLocation = si.shellInterface.GetConfigLocation()
	}

	if si.configLocation == "" {
		output, err := utils.ExecRead(context, strings.Split(si.configScript, " "))
		if err != nil {
			msg := fmt.Sprintf("Failed to retrieve configuration location %v", err)
			logrus.Errorf(msg)
			return err
		}
		si.configLocation = output[0]
	}
	var err error
	si.validator, err = si.factory.CreateValidator(si.config, si.configLocation)
	if err != nil {
		msg := fmt.Sprintf("Failed to start validator %v", err)
		logrus.Errorf(msg)
		return err
	}
	// Run prepare script
	if err := si.shellInterface.RunCmd(context, "prepare", si.prepareScript, []string{"KUBECONFIG=" + si.configLocation}); err != nil {
		return err
	}

	si.started = true

	return nil
}

func (si *shellInstance) Destroy(timeout time.Duration) error {
	logrus.Infof("Destroying cluster  %s", si.id)

	context, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return si.shellInterface.RunCmd(context, "destroy", si.stopScript, nil)
}

func (si *shellInstance) GetRoot() string {
	return si.root
}

func (si *shellInstance) doDestroy(writer *bufio.Writer, timeout time.Duration, err error) {
	_, _ = writer.WriteString(fmt.Sprintf("Error during k8s API initialisation %v", err))
	_, _ = writer.WriteString(fmt.Sprintf("Trying to destroy cluster"))
	// In case we failed to start and create cluster utils.
	err2 := si.Destroy(timeout)
	if err2 != nil {
		_, _ = writer.WriteString(fmt.Sprintf("Error during destroy of cluster %v", err2))
	}
}

func (si *shellInstance) doInstall(context context.Context) error {
	si.provider.Lock()
	defer si.provider.Unlock()
	if si.installScript != nil && !si.provider.installDone[si.config.Name] {
		si.provider.installDone[si.config.Name] = true
		return si.shellInterface.RunCmd(context, "install", si.installScript, nil)
	}
	return nil
}

func (p *shellProvider) getProviderId(provider string) string {
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
	id := fmt.Sprintf("%s-%s", config.Name, p.getProviderId(config.Name))

	root := path.Join(p.root, id)

	clusterInstance := &shellInstance{
		manager:            manager,
		provider:           p,
		root:               root,
		id:                 id,
		config:             config,
		configScript:       config.Scripts[ConfigScript],
		installScript:      utils.ParseScript(config.Scripts[InstallScript]),
		startScript:        utils.ParseScript(config.Scripts[StartScript]),
		prepareScript:      utils.ParseScript(config.Scripts[PrepareScript]),
		stopScript:         utils.ParseScript(config.Scripts[StopScript]),
		zoneSelectorScript: config.Scripts[ZoneSelector],
		factory:            factory,
		shellInterface:     shell.NewShellInterface(manager, id, root, config, instanceOptions),
		params:             instanceOptions,
	}

	return clusterInstance, nil
}

func init() {
	logrus.Infof("Adding shell as supported providers...")
	providers.ClusterProviderFactories["shell"] = NewShellClusterProvider
}

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
	if _, ok := config.Scripts[StopScript]; !ok {
		return fmt.Errorf("Invalid shutdown script location")
	}

	for _, envVar := range config.EnvCheck {
		envValue := os.Getenv(envVar)
		if envValue == "" {
			return fmt.Errorf("Environment variable are not specified %s Required variables: %v", envValue, config.EnvCheck)
		}
	}

	return nil
}
