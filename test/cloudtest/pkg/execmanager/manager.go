package execmanager

import (
	"fmt"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/utils"
	"github.com/sirupsen/logrus"
	"os"
	"path"
	"path/filepath"
	"sync"
)

type ExecutionManager interface {
	AddTestLog(category, testName, operation, content string)
	OpenFileTest(category, testname, operation string) (string, *os.File, error)
	AddLog(category, operationName, content string)
	OpenFile(category, operationName string) (string, *os.File, error)
	GetRoot(root string) (string,error)
	AddFile(fileName string, bytes []byte)
}

type executionManagerImpl struct {
	root string
	steps map[string]int
	sync.Mutex
}

// write file 'clusters/GKE/create'
// write file 'clusters/GKE/tests/testname/output'
// write file 'clusters/GKE/tests/testname/kubectl_logs'
func (mgr *executionManagerImpl) AddTestLog(category, testName, operation, content string) {
	cat := mgr.getCategory(category)
	utils.WriteFile(path.Join(mgr.root, category), fmt.Sprintf("%s-%s-%s-%s.log", cat, testName, operation), content)
}

func (mgr *executionManagerImpl) getCategory(category string) string {
	mgr.Lock()
	defer mgr.Unlock()
	val, ok := mgr.steps[category]
	if ok {
		val++
	} else {
		val = 1
	}
	mgr.steps[category] = val
	return fmt.Sprintf("%d", val)
}

func (mgr *executionManagerImpl) AddFile(fileName string, bytes []byte) {
	fileName, f, err := utils.OpenFile(mgr.root, fileName)

	if err != nil {
		logrus.Errorf("Failed to write file: %s %v", fileName, err)
		return
	}
	_, err = f.Write(bytes)
	if err != nil {
		logrus.Errorf("Failed to write content to file, %v", err)
	}
	_ = f.Close()
}

func (mgr *executionManagerImpl) OpenFile(category, operationName string) (string, *os.File, error) {
	cat := mgr.getCategory(category)
	return utils.OpenFile(path.Join(mgr.root, category), fmt.Sprintf("%s-%s.log", cat, operationName))
}
func (mgr *executionManagerImpl) OpenFileTest(category, testName, operation string) (string, *os.File, error) {
	cat := mgr.getCategory(category)
	return utils.OpenFile(path.Join(mgr.root, category), fmt.Sprintf("%s-%s-%s.log", cat, testName, operation))
}

func (mgr *executionManagerImpl) AddLog(category, operationName, content string) {
	cat := mgr.getCategory(category)

	utils.WriteFile(path.Join(mgr.root, category), fmt.Sprintf("%s-%s.log", cat, operationName), content)
}

func (mgr *executionManagerImpl) GetRoot(root string) (string, error) {
	mgr.Lock()
	defer mgr.Unlock()
	initPath := path.Join(mgr.root, root)
	if !utils.FileExists(initPath) {
		utils.CreateFolders(initPath)
		return filepath.Abs(initPath)
	} else {
		index := 2
		for {
			initPath := path.Join(mgr.root, fmt.Sprintf("%s-%d", root, index))
			if !utils.FileExists(initPath) {
				utils.CreateFolders(initPath)
				return filepath.Abs(initPath)
			}
			index++
		}
	}
}

func NewExecutionManager(root string) ExecutionManager {
	utils.ClearFolder(root, true)
	return &executionManagerImpl{
		root: root,
		steps: map[string]int{},
	}
}
