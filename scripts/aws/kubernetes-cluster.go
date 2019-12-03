package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
)

const requestInterval = 5 * time.Second

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
		createAWSKubernetesCluster()
	case "Delete":
		err := deleteAWSKubernetesCluster(os.Getenv("NSM_AWS_SERVICE_SUFFIX"))
		if err != nil {
			os.Exit(1)
		}
	case "DeleteAll":
		var durationHours int64
		var err error
		if len(os.Args) < 3 {
			durationHours = 0
		} else {
			durationHours, err = strconv.ParseInt(os.Args[2], 10, 64)
		}
		if err != nil {
			logrus.Errorf("Cannot parse: %v", err)
			return
		}
		deleteAllKubernetesClusters(time.Duration(durationHours) * time.Hour)
	default:
		printUsage()
	}
}
