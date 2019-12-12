package main

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
)

const requestInterval = 5 * time.Second

const (
	awsClusterPrefix      = "nsm"
	awsRolePrefix         = "nsm-role"
	awsClusterStackPrefix = "nsm-srv"
	awsKeyPairPrefix      = "nsm-key-pair"
	awsNodesStackPrefix   = "nsm-nodes"
	awsNodeGroupPrefix    = "nsm-node-group"
)

// AWSCluster - controlling aws clusters
type AWSCluster struct {
	configPath    string
	deferError    error
	serviceSuffix string
}

// NewAWSCluster - Creates new instance of AWS cluster for creation and deletion
func NewAWSCluster(serviceSuffix string) *AWSCluster {
	_, currentFilePath, _, ok := runtime.Caller(0)
	if !ok {
		currentFilePath = "."
	}
	return &AWSCluster{
		configPath:    path.Dir(currentFilePath),
		serviceSuffix: serviceSuffix,
		deferError:    nil,
	}
}

func printUsage() {
	fmt.Printf("Usage: go run ./... <command>\n" +
		"AWS support commands:\n" +
		"	Create			Create EKS cluster and configure kubernetes\n" +
		"	Delete			Destroy EKS cluster\n" +
		"	DeleteAll N		Destroy All EKS clusters older than N hours (Example: DeleteAll 24) \n")
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	switch os.Args[1] {
	case "Create":
		NewAWSCluster(os.Getenv("NSM_AWS_SERVICE_SUFFIX")).CreateAWSKubernetesCluster()
	case "Delete":
		err := NewAWSCluster(os.Getenv("NSM_AWS_SERVICE_SUFFIX")).DeleteAWSKubernetesCluster()
		if err != nil {
			os.Exit(1)
		}
	case "DeleteAll":
		var durationHours int64
		var namePattern string
		var err error
		if len(os.Args) < 4 {
			namePattern = ".*"
		} else {
			namePattern = os.Args[3]
		}
		if len(os.Args) < 3 {
			durationHours = 0
		} else {
			durationHours, err = strconv.ParseInt(os.Args[2], 10, 64)
		}
		if err != nil {
			logrus.Errorf("Cannot parse: %v", err)
			return
		}
		DeleteAllKubernetesClusters(time.Duration(durationHours)*time.Hour, namePattern)
	default:
		printUsage()
	}
}
