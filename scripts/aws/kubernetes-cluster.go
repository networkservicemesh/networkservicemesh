package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"runtime"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"
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

func makeCreateClusterAttempt(cluster *AWSCluster) (resultErr error) {
	defer func() {
		rErr := recover()
		if err, ok := rErr.(error); ok {
			resultErr = err
		}
		if err, ok := rErr.(awserr.Error); ok {
			resultErr = err
		}
	}()

	cluster.CreateAWSKubernetesCluster()
	return nil
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	switch os.Args[1] {
	case "Create":
		cluster := NewAWSCluster(os.Getenv("NSM_AWS_SERVICE_SUFFIX"))
		for {
			err := makeCreateClusterAttempt(cluster)
			if aerr, ok := err.(awserr.Error); ok {
				if aerr.Code() == "Throttling" {
					log.Printf("Warning (%s): %s\n", aerr.Code(), aerr.Message())
					log.Printf("Restarting AWS kubernetes cluster creation...")
					continue
				}
				log.Fatalf("Error (%s): %s\n", aerr.Code(), aerr.Message())
			} else if err != nil {
				log.Fatalf("Error: %s\n", err.Error())
			}
			break
		}
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
