package commands

import (
	"context"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/config"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/execmanager"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/utils"
	"github.com/sirupsen/logrus"
	"strings"
	"time"
)

type Status int8

const (
	// Success execution on all clusters
	Status_ADDED               Status = 0 // Just added
	Status_SUCCESS             Status = 1 // Passed
	Status_FAILED              Status = 2 // Failed execution on all clusters
	Status_TIMEOUT             Status = 3 // Test timeout waiting for results
	Status_SKIPPED             Status = 4
	Status_SKIPPED_NO_CLUSTERS Status = 5
)

type TestEntryExecution struct {
	OutputFile string // Output file name
	retry      int    // Did we retry execution on this cluster.
	Status     Status // Execution status
}
type TestEntry struct {
	Name            string // Test name
	Tags            string // A list of tags
	ExecutionConfig *config.ExecutionConfig
	Status          Status

	Executions []TestEntryExecution
	Duration   time.Duration
	Started    time.Time
}

// Return list of available tests by calling of gotest --list .* $root -tag "" and parsing of output.
func GetTestConfiguration(manager execmanager.ExecutionManager, root string, tags []string) ([]*TestEntry, error) {
	gotestCmd := []string{"go", "test", root, "--list", ".*"}
	if len(tags) > 0 {
		result := []*TestEntry{}
		tagsStr := strings.Join(tags, " ")
		tests, err := getTests(manager, append(gotestCmd, "-tags", tagsStr), tagsStr)
		if err != nil {
			return nil, err
		}
		logrus.Infof("Found %d tests with tags %s", len(tests), tagsStr)
		result = append(result, tests...)
		return result, nil
	} else {
		return getTests(manager, gotestCmd, "")
	}
}

func getTests(manager execmanager.ExecutionManager, gotestCmd []string, tag string) ([]*TestEntry, error) {
	result, err := utils.ExecRead(context.Background(), gotestCmd)
	if err != nil {
		logrus.Errorf("Error getting list of tests %v", err)
	}

	var testResult []*TestEntry

	manager.AddLog("gotest", "find-tests", strings.Join(gotestCmd, " ")+"\n"+strings.Join(result, "\n"))
	for _, testLine := range result {
		if strings.ContainsAny(testLine, "\t") {
			special := strings.Split(testLine, "\t")
			if len(special) == 3 {
				// This is special case.
			}
		} else {
			testResult = append(testResult, &TestEntry{
				Name: strings.TrimSpace(testLine),
				Tags: tag,
			})
		}
	}
	return testResult, nil
}
