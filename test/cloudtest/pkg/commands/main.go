package commands

import (
	"bufio"
	"context"
	"encoding/xml"
	"fmt"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/config"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/execmanager"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/k8s"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/providers"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/providers/packet"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/providers/shell"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/reporting"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	defaultConfigFile string = ".cloudtest.yaml"
)

// Arguments - command line arguments
type Arguments struct {
	clusters        []string // A list of enabled clusters from configuration.
	providerConfig  string   // A folder to start scaning for tests inside
	count           int      // Limit number of tests to be run per every cloud
	instanceOptions providers.InstanceOptions
	onlyEnabled     bool // Disable all clusters and enable only enabled in command line.
}

type clusterState byte

const (
	clusterAdded        clusterState = 0
	clusterReady        clusterState = 1
	clusterBusy         clusterState = 2
	clusterStarting     clusterState = 3
	clusterCrashed      clusterState = 4
	clusterNotAvailable clusterState = 5
)

type clusterInstance struct {
	instance      providers.ClusterInstance
	state         clusterState
	startCount    int
	id            string
	taskCancel    context.CancelFunc
	cancelMonitor context.CancelFunc
	startTime     time.Time
	lock          sync.Mutex
}
type clustersGroup struct {
	instances []*clusterInstance
	provider  providers.ClusterProvider
	config    *config.ClusterProviderConfig
	tasks     []*testTask // All tasks assigned to this cluster.
}

type testTask struct {
	taskID           string
	test             *TestEntry
	cluster          *clustersGroup
	clusterInstances []*clusterInstance
	clusterTaskID    string
}

type eventKind byte

const (
	eventTaskUpdate    eventKind = 0
	eventClusterUpdate eventKind = 1
)

type operationEvent struct {
	kind            eventKind
	cluster         *clustersGroup
	clusterInstance *clusterInstance
	task            *testTask
}

type executionContext struct {
	manager          execmanager.ExecutionManager
	clusters         []*clustersGroup
	operationChannel chan operationEvent
	tests            []*TestEntry
	tasks            []*testTask
	running          map[string]*testTask
	completed        []*testTask
	skipped          []*testTask
	cloudTestConfig  *config.CloudTestConfig
	report           *reporting.JUnitFile
	startTime        time.Time
	clusterReadyTime time.Time
	factory          k8s.ValidationFactory
	arguments        *Arguments
}

// CloudTestRun - CloudTestRun
func CloudTestRun(cmd *cloudTestCmd) {
	var configFileContent []byte
	var err error

	if cmd.cmdArguments.providerConfig == "" {
		cmd.cmdArguments.providerConfig = defaultConfigFile
	}

	configFileContent, err = ioutil.ReadFile(cmd.cmdArguments.providerConfig)
	if err != nil {
		logrus.Errorf("Failed to read config file %v", err)
		return
	}

	// Root config
	testConfig := &config.CloudTestConfig{}
	err = parseConfig(testConfig, configFileContent)
	if err != nil {
		os.Exit(1)
	}

	// Process config imports
	err = performImport(testConfig)
	if err != nil {
		logrus.Errorf("Failed to process config imports %v", err)
		os.Exit(1)
	}


	_, err = PerformTesting(testConfig, k8s.CreateFactory(), cmd.cmdArguments)
	if err != nil {
		os.Exit(1)
	}
}

func performImport(testConfig *config.CloudTestConfig) error {
	for _, imp := range testConfig.Imports {
		importConfig := &config.CloudTestConfig{}

		configFileContent, err := ioutil.ReadFile(imp)
		if err != nil {
			logrus.Errorf("Failed to read config file %v", err)
			return err
		}
		if err = parseConfig(importConfig, configFileContent); err != nil {
			return err
		}

		// Do add imported items
		testConfig.Executions = append(testConfig.Executions, importConfig.Executions...)
		testConfig.Providers = append(testConfig.Providers, importConfig.Providers...)
	}
	return nil
}

// PerformTesting - PerformTesting
func PerformTesting(config *config.CloudTestConfig, factory k8s.ValidationFactory, arguments *Arguments) (*reporting.JUnitFile, error) {
	ctx := &executionContext{
		cloudTestConfig:  config,
		operationChannel: make(chan operationEvent),
		tasks:            []*testTask{},
		running:          map[string]*testTask{},
		completed:        []*testTask{},
		tests:            []*TestEntry{},
		factory:          factory,
		arguments:        arguments,
	}
	ctx.manager = execmanager.NewExecutionManager(ctx.cloudTestConfig.ConfigRoot)
	// Create cluster instance handles
	if err := ctx.createClusters(); err != nil {
		return nil, err
	}
	// Collect tests
	ctx.findTests()
	// We need to be sure all clusters will be deleted on end of execution.
	defer ctx.performShutdown()
	// Fill tasks to be executed..
	ctx.createTasks()

	ctx.performExecution()

	return ctx.generateJUnitReportFile()
}

func parseConfig(cloudTestConfig *config.CloudTestConfig, configFileContent []byte) error {
	err := yaml.Unmarshal(configFileContent, cloudTestConfig)
	if err != nil {
		err = fmt.Errorf("Failed to parse configuration file: %v", err)
		logrus.Errorf(err.Error())
		return err
	}
	logrus.Infof("Configuration file loaded successfully...")
	return nil
}

func (ctx *executionContext) performShutdown() {
	// We need to stop all clusters we started
	if !ctx.arguments.instanceOptions.NoStop {
		var wg sync.WaitGroup
		for _, clG := range ctx.clusters {
			group := clG
			for _, cInst := range group.instances {
				curInst := cInst
				logrus.Infof("Schedule Closing cluster %v %v", group.config.Name, curInst.id)
				wg.Add(1)

				go func() {
					defer wg.Done()
					logrus.Infof("Closing cluster %v %v", group.config.Name, curInst.id)
					ctx.destroyCluster(group, curInst, false)
				}()
			}
		}
		wg.Wait()
	}
	logrus.Infof("All clusters destroyed")
}

func (ctx *executionContext) performExecution() {
	logrus.Infof("Starting test execution")
	ctx.startTime = time.Now()
	ctx.clusterReadyTime = ctx.startTime

	termChannel := tools.NewOSSignalChannel()
	terminated := false
	for len(ctx.tasks) > 0 || len(ctx.running) > 0 {
		// WE take 1 test task from list and do execution.
		if terminated {
			break
		}
		ctx.assignTasks(func() bool { return terminated })

		select {
		case event := <-ctx.operationChannel:
			switch event.kind {
			case eventClusterUpdate:
				ctx.performClusterUpdate(event)
				ctx.printStatistics()
			case eventTaskUpdate:
				// Remove from running onces.
				ctx.processTaskUpdate(event)
			}
		case <-time.After(1 * time.Minute):
			ctx.printStatistics()
		case <-termChannel:
			logrus.Errorf("Termination request is received")
			terminated = true
		}
	}
	logrus.Infof("Completed tasks %v", len(ctx.completed))
}

func (ctx *executionContext) assignTasks(terminated func() bool) {
	if len(ctx.tasks) > 0 {
		// Lets check if we have cluster required and start it
		// Check if we have cluster we could assign.
		newTasks := []*testTask{}
		for _, task := range ctx.tasks {
			if terminated() {
				break
			}
			clustersAvailable, clustersToUse, assigned := ctx.selectClusterForTask(task, terminated)
			if assigned {
				err := ctx.startTask(task, clustersToUse)
				if err != nil {
					logrus.Errorf("Error starting task  %s %v", task.test.Name, err)
					assigned = false
				} else {
					ctx.running[task.taskID] = task
				}
			}
			// If we finally not assigned.
			if !assigned {
				if clustersAvailable == 0 {
					// We move task to skipped since, no clusters could execute it, all attempts for clusters to recover are finished.
					task.test.Status = statusSkippedSinceNoClusters
					ctx.completed = append(ctx.completed, task)
				} else {
					newTasks = append(newTasks, task)
				}
			}
		}
		ctx.tasks = newTasks
	}
}

func (ctx *executionContext) performClusterUpdate(event operationEvent) {
	logrus.Infof("Instance for cluster %s is updated %v", event.cluster.config.Name, event.clusterInstance)
	if event.clusterInstance.taskCancel != nil && event.clusterInstance.state == clusterCrashed {
		// We have task running on cluster
		event.clusterInstance.taskCancel()
	}
	if event.clusterInstance.state == clusterReady {
		if ctx.clusterReadyTime == ctx.startTime {
			ctx.clusterReadyTime = time.Now()
		}
	}
}

func (ctx *executionContext) processTaskUpdate(event operationEvent) {
	delete(ctx.running, event.task.taskID)
	// Make cluster as ready
	for _, inst := range event.task.clusterInstances {
		ctx.setClusterState(inst, func(inst *clusterInstance) {
			if inst.state != clusterCrashed {
				inst.state = clusterReady
			}
			inst.taskCancel = nil
		})
	}
	if event.task.test.Status == statusSuccess || event.task.test.Status == statusFailed {
		ctx.completed = append(ctx.completed, event.task)

		elapsed := time.Since(ctx.startTime)
		oneTask := elapsed / time.Duration(len(ctx.completed))
		logrus.Infof("Complete task %s on cluster %s, Elapsed: %v (%d) Remaining: %v (%d)",
			event.task.test.Name, event.task.clusterTaskID, elapsed,
			len(ctx.completed),
			time.Duration(len(ctx.tasks)+len(ctx.running))*oneTask,
			len(ctx.running)+len(ctx.tasks))
	} else {
		logrus.Infof("Re schedule task %v", event.task.test.Name)
		ctx.tasks = append(ctx.tasks, event.task)
	}
}

func (ctx *executionContext) selectClusterForTask(task *testTask, terminationCheck func() bool) (int, []*clusterInstance, bool) {
	var clustersToUse []*clusterInstance
	assigned := false
	clustersAvailable := 0
	for _, ci := range task.cluster.instances {
		if terminationCheck() {
			break
		}
		ciref := ci
		// No task is assigned for cluster.
		switch ciref.state {
		case clusterAdded, clusterCrashed:
			// Try starting cluster
			ctx.startCluster(task.cluster, ciref)
			clustersAvailable++
		case clusterReady:
			// Check if we match requirements.
			// We could assign task and start it running.
			clustersToUse = append(clustersToUse, ciref)
			// We need to remove task from list
			assigned = true
			clustersAvailable++
		case clusterBusy, clusterStarting:
			clustersAvailable++
		}
		if assigned {
			// Task is scheduled
			break
		}
	}
	return clustersAvailable, clustersToUse, assigned
}

func (ctx *executionContext) printStatistics() {
	elapsed := time.Since(ctx.startTime)
	elapsedRunning := time.Since(ctx.clusterReadyTime)
	running := ""
	for _, r := range ctx.running {
		running += fmt.Sprintf("\t\t%s on cluster %v elapsed: %v\n", r.test.Name, r.clusterTaskID, time.Since(r.test.Started))
	}
	if len(running) > 0 {
		running = "\n\tRunning:\n" + running
	}
	clustersMsg := ""

	for _, cl := range ctx.clusters {
		clustersMsg += fmt.Sprintf("\t\tCluster: %v\n", cl.config.Name)
		for _, inst := range cl.instances {
			clustersMsg += fmt.Sprintf("\t\t\t%s %v uptime: %v\n", inst.id, fromClusterState(inst.state),
				time.Since(inst.startTime))
		}
	}
	if len(clustersMsg) > 0 {
		clustersMsg = "\n\tClusters:\n" + clustersMsg
	}
	remaining := ""
	if len(ctx.completed) > 0 {
		oneTask := elapsed / time.Duration(len(ctx.completed))
		remaining = fmt.Sprintf("%v", time.Duration(len(ctx.tasks)+len(ctx.running))*oneTask)
	}
	logrus.Infof("Statistics:"+
		"\n\tElapsed total: %v"+
		"\n\tTests time: %v"+
		"\n\tTasks  Completed: %d"+
		"\n\t		Remaining: %v (%d).\n"+
		"%s%s",
		elapsed,
		elapsedRunning, len(ctx.completed),
		remaining, len(ctx.running)+len(ctx.tasks),
		running, clustersMsg)
}

func fromClusterState(state clusterState) string {
	switch state {
	case clusterReady:
		return "ready"
	case clusterAdded:
		return "added"
	case clusterBusy:
		return "running test"
	case clusterCrashed:
		return "crashed"
	case clusterNotAvailable:
		return "not available"
	case clusterStarting:
		return "starting"
	}
	return fmt.Sprintf("unknown state: %v", state)
}

func (ctx *executionContext) createTasks() {
	taskIndex := 0
	for i, test := range ctx.tests {
		for _, cluster := range ctx.clusters {
			if (len(test.ExecutionConfig.ClusterSelector) > 0 && utils.Contains(test.ExecutionConfig.ClusterSelector, cluster.config.Name)) ||
				len(test.ExecutionConfig.ClusterSelector) == 0 {
				// Cluster selector is defined we need to add tasks for individual cluster only
				task := &testTask{
					taskID: fmt.Sprintf("%d", taskIndex),
					test: &TestEntry{
						Name:            test.Name,
						Tags:            test.Tags,
						Status:          test.Status,
						ExecutionConfig: test.ExecutionConfig,
						Executions:      []TestEntryExecution{},
					},
					cluster: cluster,
				}
				taskIndex++
				cluster.tasks = append(cluster.tasks, task)

				if ctx.arguments.count > 0 && i >= ctx.arguments.count {
					logrus.Infof("Limit of tests for execution:: %v is reached. Skipping test %s", ctx.arguments.count, test.Name)
					test.Status = statusSkipped
					ctx.skipped = append(ctx.skipped, task)
				} else {
					ctx.tasks = append(ctx.tasks, task)
				}
			}
		}
	}
}

func (ctx *executionContext) startTask(task *testTask, instances []*clusterInstance) error {
	ids := ""
	for _, ci := range instances {
		if len(ids) > 0 {
			ids += "_"
		}
		ids += ci.id

		ctx.setClusterState(ci, func(ci *clusterInstance) {
			ci.state = clusterBusy
		})
	}
	fileName, file, err := ctx.manager.OpenFileTest(ids, task.test.Name, "run")
	if err != nil {
		return err
	}

	clusterConfigs := []string{}

	for _, inst := range instances {
		clusterConfig, err := inst.instance.GetClusterConfig()
		if err != nil {
			return err
		}
		clusterConfigs = append(clusterConfigs, clusterConfig)
	}

	task.clusterInstances = instances
	task.clusterTaskID = ids

	timeout := ctx.getTestTimeout(task)

	go func() {
		st := time.Now()
		cmdLine := []string{
			"go", "test",
			task.test.ExecutionConfig.PackageRoot,
			"-test.timeout", fmt.Sprintf("%ds", timeout),
			"-count", "1",
			"--run", fmt.Sprintf("^(%s)$", task.test.Name),
			"--tags", task.test.Tags,
			"--test.v",
		}

		env := []string{
		}
		// Fill Kubernetes environment variables.

		for ind, envV := range task.test.ExecutionConfig.KubernetesEnv {
			env = append(env, fmt.Sprintf("%s=%s", envV, clusterConfigs[ind]))
		}

		writer := bufio.NewWriter(file)

		timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout*2)*time.Second)
		defer cancel()

		logrus.Infof(fmt.Sprintf("Running test %s on cluster's %v \n", task.test.Name, ids))

		_, _ = writer.WriteString(fmt.Sprintf("Running test %s on cluster's %v \n", task.test.Name, ids))
		_, _ = writer.WriteString(fmt.Sprintf("Command line %v env==%v \n", cmdLine, env))
		_ = writer.Flush()

		for _, inst := range instances {
			inst.taskCancel = cancel
		}

		task.test.Started = time.Now()
		proc, err := utils.ExecProc(timeoutCtx, cmdLine, env)
		if err != nil {
			logrus.Errorf("Failed to run %s %v", cmdLine, err)
			ctx.updateTestExecution(task, fileName, statusFailed)
		}
		go func() {
			reader := bufio.NewReader(proc.Stdout)
			for {
				s, err := reader.ReadString('\n')
				if err != nil {
					break
				}
				_, _ = writer.WriteString(s)
				_ = writer.Flush()
			}
		}()
		code := proc.ExitCode()
		task.test.Duration = time.Since(st)

		if code != 0 {
			// Check if cluster is alive.
			clusterNotAvailable := false
			for _, inst := range instances {
				err := inst.instance.CheckIsAlive()
				if err != nil {
					clusterNotAvailable = true
					ctx.destroyCluster(task.cluster, inst, true)
				}
				inst.taskCancel = nil
			}

			if timeoutCtx.Err() == context.Canceled || clusterNotAvailable {
				logrus.Errorf("Test is canceled due timeout or cluster error.. Will be re-run")
				ctx.updateTestExecution(task, fileName, statusTimeout)
			} else {
				msg := fmt.Sprintf("Failed to run %s Exit code: %v. Logs inside %v \n", cmdLine, code, fileName)
				logrus.Errorf(msg)
				_, _ = writer.WriteString(msg)
				_ = writer.Flush()
				ctx.updateTestExecution(task, fileName, statusFailed)
			}
		} else {
			ctx.updateTestExecution(task, fileName, statusSuccess)
		}
	}()
	return nil
}

func (ctx *executionContext) getTestTimeout(task *testTask) int64 {
	timeout := task.test.ExecutionConfig.Timeout
	if timeout == 0 {
		logrus.Infof("test timeout is not specified, use default value, 3min")
		timeout = 3 * 60
	}
	return timeout
}

func (ctx *executionContext) updateTestExecution(task *testTask, fileName string, status Status) {
	task.test.Status = status
	task.test.Executions = append(task.test.Executions, TestEntryExecution{
		Status:     status,
		retry:      len(task.test.Executions) + 1,
		OutputFile: fileName,
	})
	ctx.operationChannel <- operationEvent{
		cluster: task.cluster,
		task:    task,
		kind:    eventTaskUpdate,
	}
}

func (ctx *executionContext) startCluster(group *clustersGroup, ci *clusterInstance) {
	ci.lock.Lock()
	defer ci.lock.Unlock()

	if ci.state != clusterAdded && ci.state != clusterCrashed {
		// Cluster is already starting.
		return
	}

	if ci.startCount > group.config.RetryCount {
		ci.state = clusterNotAvailable
		return
	}

	ci.state = clusterStarting
	go func() {
		timeout := ctx.getClusterTimeout(group)
		ci.startCount++
		err := ci.instance.Start(timeout)
		if err != nil {
			ctx.destroyCluster(group, ci, true)
			ctx.setClusterState(ci, func(ci *clusterInstance) {
				ci.state = clusterCrashed
			})
		}
		// Starting cloud monitoring thread
		if ci.state != clusterCrashed {
			monitorContext, monitorCancel := context.WithCancel(context.Background())
			ci.cancelMonitor = monitorCancel
			ctx.monitorCluster(monitorContext, ci, group)
		} else {
			ctx.operationChannel <- operationEvent{
				kind:            eventClusterUpdate,
				cluster:         group,
				clusterInstance: ci,
			}
		}
	}()
}

func (ctx *executionContext) getClusterTimeout(group *clustersGroup) time.Duration {
	timeout := time.Duration(group.config.Timeout) * time.Second
	if group.config.Timeout == 0 {
		logrus.Infof("test timeout is not specified, use default value 5min")
		timeout = 5 * time.Minute
	}
	return timeout
}

func (ctx *executionContext) monitorCluster(context context.Context, ci *clusterInstance, group *clustersGroup) {
	checks := 0
	for {
		err := ci.instance.CheckIsAlive()
		if err != nil {
			logrus.Errorf("Failed to interact with cluster %v", ci.id)
			ctx.destroyCluster(group, ci, true)
			break
		}

		if checks == 0 {
			// Initial check performed, we need to make cluster ready.
			ctx.setClusterState(ci, func(ci *clusterInstance) {
				ci.state = clusterReady
				ci.startTime = time.Now()
			})
			ctx.operationChannel <- operationEvent{
				kind:            eventClusterUpdate,
				cluster:         group,
				clusterInstance: ci,
			}
			logrus.Infof("cluster started...")
		}
		checks++;
		select {
		case <-time.After(5 * time.Second):
			// Just pass
		case <-context.Done():
			logrus.Infof("cluster monitoring is canceled: %s. Uptime: %v seconds", ci.id, checks*5)
			return
		}
	}
}

func (ctx *executionContext) destroyCluster(group *clustersGroup, ci *clusterInstance, sendUpdate bool) {
	ci.lock.Lock()
	defer ci.lock.Unlock()

	if ci.cancelMonitor != nil {
		ci.cancelMonitor()
	}

	if ci.state == clusterCrashed || ci.state == clusterNotAvailable {
		// It is already destroyed or not available.
		return
	}

	ci.state = clusterBusy

	timeout := ctx.getClusterTimeout(group)
	err := ci.instance.Destroy(timeout)
	if err != nil {
		logrus.Errorf("Failed to destroy cluster")
	}

	if group.config.StopDelay != 0 {
		logrus.Infof("Cluster stop wormup timeout specified %v", group.config.StopDelay)
		<-time.After(time.Duration(group.config.StopDelay) * time.Second)
	}
	ci.state = clusterCrashed
	if sendUpdate {
		ctx.operationChannel <- operationEvent{
			cluster:         group,
			clusterInstance: ci,
			kind:            eventClusterUpdate,
		}
	}

}

func (ctx *executionContext) createClusters() error {
	ctx.clusters = []*clustersGroup{}
	clusterProviders, err := createClusterProviders(ctx.manager)
	if err != nil {
		return err
	}

	for _, cl := range ctx.cloudTestConfig.Providers {
		if ctx.arguments.onlyEnabled {
			logrus.Infof("Disable cluster config:: %v since onlyEnabled is passed...", cl.Name)
			cl.Enabled = false
		}
		for _, cc := range ctx.arguments.clusters {
			if cl.Name == cc {
				if !cl.Enabled {
					logrus.Infof("Enabling config:: %v", cl.Name)
				}
				cl.Enabled = true
			}
		}
		if cl.Enabled {
			logrus.Infof("Initialize provider for config:: %v %v", cl.Name, cl.Kind)
			provider, ok := clusterProviders[cl.Kind]
			if !ok {
				msg := fmt.Sprintf("Cluster provider %s are not found...", cl.Kind)
				logrus.Errorf(msg)
				return fmt.Errorf(msg)
			}
			instances := []*clusterInstance{}
			for i := 0; i < cl.Instances; i++ {
				cluster, err := provider.CreateCluster(cl, ctx.factory, ctx.manager, ctx.arguments.instanceOptions)
				if err != nil {
					msg := fmt.Sprintf("Failed to create cluster instance. Error %v", err)
					logrus.Errorf(msg)
					return fmt.Errorf(msg)
				}
				instances = append(instances, &clusterInstance{
					instance:  cluster,
					startTime: time.Now(),
					state:     clusterAdded,
					id:        cluster.GetID(),
				})
			}
			if len(instances) == 0 {
				msg := fmt.Sprintf("No instances are specified for %s.", cl.Name)
				logrus.Errorf(msg)
				return fmt.Errorf(msg)
			}
			ctx.clusters = append(ctx.clusters, &clustersGroup{
				provider:  provider,
				instances: instances,
				config:    cl,
			})
		}
	}
	if len(ctx.clusters) == 0 {
		msg := "there is no clusters defined. Exiting"
		logrus.Errorf(msg)
		return fmt.Errorf(msg)
	}
	return nil
}

func (ctx *executionContext) findTests() {
	logrus.Infof("Finding tests")
	testsMap := map[string]*TestEntry{}
	for _, exec := range ctx.cloudTestConfig.Executions {
		st := time.Now()
		execTests, err := GetTestConfiguration(ctx.manager, exec.PackageRoot, exec.Tags)
		if err != nil {
			logrus.Errorf("Failed during test lookup %v", err)
		}
		logrus.Infof("Tests found: %v Elapsed: %v", len(execTests), time.Since(st))
		for _, t := range execTests {
			t.ExecutionConfig = exec
			if existing, ok := testsMap[t.Name]; ok {
				// Test is already added
				if existing.Tags != "" && t.Tags == "" {
					// test without tags are in priority
					testsMap[t.Name] = t
				}
			} else {
				testsMap[t.Name] = t
			}
		}
	}
	// If we have execution without tags, we need to remove all tests from it from tagged executions.

	for _, v := range testsMap {
		v.Status = statusAdded
		ctx.tests = append(ctx.tests, v)
	}

	logrus.Infof("Total tests found: %v", len(ctx.tests))
	if len(ctx.tests) == 0 {
		logrus.Errorf("There is no tests defined. Exiting...")
	}
}

func (ctx *executionContext) generateJUnitReportFile() (*reporting.JUnitFile, error) {
	// generate and write report
	ctx.report = &reporting.JUnitFile{
	}

	totalFailures := 0
	for _, cluster := range ctx.clusters {
		failures := 0
		totalTests := 0
		totalTime := time.Duration(0)
		suite := &reporting.Suite{
			Name: cluster.config.Name,
		}

		for _, test := range cluster.tasks {
			testCase := &reporting.TestCase{
				Name: test.test.Name,
				Time: fmt.Sprintf("%v", test.test.Duration),
			}
			totalTests++
			totalTime += test.test.Duration

			switch test.test.Status {
			case statusFailed, statusTimeout:
				failures++

				message := fmt.Sprintf("Test execution failed %v", test.test.Name)
				result := ""
				for _, ex := range test.test.Executions {
					lines, err := utils.ReadFile(ex.OutputFile)
					if err != nil {
						logrus.Errorf("Failed to read stored output %v", ex.OutputFile)
						lines = []string{"Failed to read stored output:", ex.OutputFile, err.Error()}
					}
					result = strings.Join(lines, "\n")
				}

				testCase.Failure = &reporting.Failure{
					Type:     "ERROR",
					Contents: result,
					Message:  message,
				}
			case statusSkipped:
				testCase.SkipMessage = &reporting.SkipMessage{
					Message: "By limit of number of tests to run",
				}
			case statusSkippedSinceNoClusters:
				testCase.SkipMessage = &reporting.SkipMessage{
					Message: "No clusters are available, all clusters reached restart limits...",
				}
			}
			suite.TestCases = append(suite.TestCases, testCase)
		}
		suite.Tests = totalTests
		suite.Failures = failures
		suite.Time = fmt.Sprintf("%v", totalTime)
		totalFailures += failures

		ctx.report.Suites = append(ctx.report.Suites, suite)
	}

	output, err := xml.MarshalIndent(ctx.report, "  ", "    ")
	if err != nil {
		logrus.Errorf("failed to store JUnit xml report: %v\n", err)
	}
	if ctx.cloudTestConfig.Reporting.JUnitReportFile != "" {
		ctx.manager.AddFile(ctx.cloudTestConfig.Reporting.JUnitReportFile, output)
	}
	if totalFailures > 0 {
		return ctx.report, fmt.Errorf("there is failed tests %v", totalFailures)
	}
	return ctx.report, nil
}

func (ctx *executionContext) setClusterState(instance *clusterInstance, op func(cluster *clusterInstance)) {
	instance.lock.Lock()
	defer instance.lock.Unlock()
	op(instance)
}

func createClusterProviders(manager execmanager.ExecutionManager) (map[string]providers.ClusterProvider, error) {
	clusterProviders := map[string]providers.ClusterProvider{}

	clusterProviderFactories := map[string]providers.ClusterProviderFunction{
		"packet": packet.NewPacketClusterProvider,
		"shell":  shell.NewShellClusterProvider,
	}

	for key, factory := range clusterProviderFactories {
		if _, ok := clusterProviders[key]; ok {
			msg := fmt.Sprintf("Re-definition of cluster provider... Exiting")
			logrus.Errorf(msg)
			return nil, fmt.Errorf(msg)
		}
		root, err := manager.GetRoot(key)
		if err != nil {
			logrus.Errorf("Failed to create cluster provider %v", err)
			return nil, err
		}
		clusterProviders[key] = factory(root)
	}
	return clusterProviders, nil
}

type cloudTestCmd struct {
	cobra.Command

	cmdArguments *Arguments
}

// ExecuteCloudTest - main entry point for command
func ExecuteCloudTest() {
	var rootCmd = &cloudTestCmd{
		cmdArguments: &Arguments{
			providerConfig: defaultConfigFile,
			clusters:       []string{},
		},
	}
	rootCmd.Use = "cloud_test"
	rootCmd.Short = "NSM Cloud Test is cloud helper continuous integration testing tool"
	rootCmd.Long = `Allow to execute all set of individual tests across all clouds provided.`
	rootCmd.Run = func(cmd *cobra.Command, args []string) {
		CloudTestRun(rootCmd)
	}
	rootCmd.Args = func(cmd *cobra.Command, args []string) error {
		return nil
	}

	initCmd(rootCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func initCmd(rootCmd *cloudTestCmd) {
	cobra.OnInitialize(initConfig)
	rootCmd.Flags().StringVarP(&rootCmd.cmdArguments.providerConfig, "config", "", "", "Config file for providers, default="+defaultConfigFile)
	rootCmd.Flags().StringArrayVarP(&rootCmd.cmdArguments.clusters, "clusters", "c", []string{}, "Enable disable cluster configs, default use from config. Cloud be used to test against selected configuration or locally...")
	rootCmd.Flags().BoolVarP(&rootCmd.cmdArguments.onlyEnabled, "enabled", "e", false, "Use only passed cluster names...")
	rootCmd.Flags().IntVarP(&rootCmd.cmdArguments.count, "count", "", -1, "Execute only count of tests")

	rootCmd.Flags().BoolVarP(&rootCmd.cmdArguments.instanceOptions.NoStop, "noStop", "", false, "Pass to disable stop operations...")
	rootCmd.Flags().BoolVarP(&rootCmd.cmdArguments.instanceOptions.NoInstall, "noInstall", "", false, "Pass to disable do install operations...")
	rootCmd.Flags().BoolVarP(&rootCmd.cmdArguments.instanceOptions.NoMaskParameters, "noMask", "", false, "Pass to disable masking of environment variables...")

	var versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print the version number of cloud_test",
		Long:  `All software has versions.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Cloud Test -- HEAD")
		},
	}
	rootCmd.AddCommand(versionCmd)
}

func initConfig() {
}
