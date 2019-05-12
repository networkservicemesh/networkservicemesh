package main

import (
	"fmt"
	"os"
	"time"
)

const requestInterval = 5 * time.Second

func printUsage() {
	fmt.Printf("Usage: go run ./... <command>\n" +
		"AWS support commands:\n" +
		"	Create		Create EKS cluster and configure kubernetes\n" +
		"	Delete		Destroy EKS cluster\n")

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
		deleteAWSKubernetesCluster()
	default:
		printUsage()
	}
}
