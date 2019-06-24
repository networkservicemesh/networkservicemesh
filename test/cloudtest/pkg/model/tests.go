package model

import (
	"context"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/config"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/execmanager"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/utils"
	"github.com/sirupsen/logrus"
	"strings"
	"time"
)

// Status - Test Execution status
type Status int8

const (
	// StatusAdded - test is added
	StatusAdded                  Status = 0 // statusAdded - Just added
	// StatusSuccess - test is completed fine.
	StatusSuccess                Status = 1 // Passed
	// StatusFailed - test is failed to be executed.
	StatusFailed                 Status = 2 // Failed execution on all clusters
	// StatusTimeout - timeout during test execution.
	StatusTimeout                Status = 3 // Test timeout waiting for results
	// StatusSkipped - status if test is marked as skipped.
	StatusSkipped                Status = 4 // Test is skipped
	// StatusSkippedSinceNoClusters - status of test if not clusters of desired group are available.
	StatusSkippedSinceNoClusters Status = 5 // Test is skipped since there is not clusters to execute on.
)

// TestEntryExecution - represent one test execution.
type TestEntryExecution struct {
	OutputFile string // Output file name
	Retry      int    // Did we retry execution on this cluster.
	Status     Status // Execution status
}

//TestEntryKind - describes a testing way.
type TestEntryKind uint8

const (
	// TestEntryKindGoTest - go test test
	TestEntryKindGoTest    TestEntryKind = 0
	// TestEntryKindShellTest - shell test.
	TestEntryKindShellTest TestEntryKind = 1
)

// TestEntry - represent one found test
type TestEntry struct {
	Name            string // Test name
	Tags            string // A list of tags
	ExecutionConfig *config.ExecutionConfig

	Executions []TestEntryExecution
	Duration   time.Duration
	Started    time.Time

	RunScript string

	Kind            TestEntryKind
	Status          Status
}

// GetTestConfiguration - Return list of available tests by calling of gotest --list .* $root -tag "" and parsing of output.
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
	}
	return getTests(manager, gotestCmd, "")
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
				continue
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
