package commands

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/utils"

	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/execmanager"

	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/model"

	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/config"

	"github.com/onsi/gomega"
)

func TestUpdateTaskWithTimeout_ShouldNotCompeteTask(t *testing.T) {
	assert := gomega.NewWithT(t)
	tmpDir, err := ioutil.TempDir(os.TempDir(), t.Name())
	defer utils.ClearFolder(tmpDir, false)
	assert.Expect(err).To(gomega.BeNil())

	ctx := executionContext{
		cloudTestConfig:  config.NewCloudTestConfig(),
		manager:          execmanager.NewExecutionManager(tmpDir),
		running:          make(map[string]*testTask),
		operationChannel: make(chan operationEvent, 1),
	}
	ctx.cloudTestConfig.Timeout = 2
	ctx.cloudTestConfig.Statistics.Enabled = false
	task := &testTask{
		test: &model.TestEntry{
			ExecutionConfig: &config.ExecutionConfig{
				Timeout: 1,
			},
			Status: model.StatusSkipped,
		},
	}
	ctx.tasks = append(ctx.tasks, task)
	ctx.updateTestExecution(task, "", model.StatusTimeout)
	_ = ctx.pollEvents(context.Background())
	assert.Expect(len(ctx.completed)).Should(gomega.BeZero())
}
