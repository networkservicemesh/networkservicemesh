package main

import (
	"log"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/iam"
)

var deferError = false

func checkDeferError(err error) bool {
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == "Throttling" {
				log.Printf("Warning (%s): %s\n", aerr.Code(), aerr.Message())
				return false
			}

			switch aerr.Code() {
			case "NoSuchEntity":
			case "ResourceNotFoundException":
			case "ValidationError":
			case "InvalidParameterValue":
			default:
				log.Printf("Error (%s): %s\n", aerr.Code(), aerr.Message())
				deferError = true
				return true
			}
			log.Printf("Warning (%s): %s\n", aerr.Code(), aerr.Message())
		} else {
			log.Printf("Error: %s\n", err.Error())
			deferError = true
		}
		return true
	}
	return false
}

func DeleteEksRole(iamClient *iam.IAM, eksRoleName *string) {
	log.Printf("Deleting EKS service role \"%s\"...\n", *eksRoleName)

	policyArn := "arn:aws:iam::aws:policy/AmazonEKSClusterPolicy"
	_, err := iamClient.DetachRolePolicy(&iam.DetachRolePolicyInput{
		RoleName:  eksRoleName,
		PolicyArn: &policyArn,
	})
	checkDeferError(err)

	policyArn = "arn:aws:iam::aws:policy/AmazonEKSServicePolicy"
	_, err = iamClient.DetachRolePolicy(&iam.DetachRolePolicyInput{
		RoleName:  eksRoleName,
		PolicyArn: &policyArn,
	})
	checkDeferError(err)

	_, err = iamClient.DeleteRole(&iam.DeleteRoleInput{
		RoleName: eksRoleName,
	})
	if checkDeferError(err) {
		return
	}

	log.Printf("Role \"%s\" successfully deleted!\n", *eksRoleName)
}

func DeleteEC2NetworkInterfaces(ec2Client *ec2.EC2, cfClient *cloudformation.CloudFormation, nodesStackName *string) {
	cfResp, err := cfClient.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: nodesStackName,
	})
	if checkDeferError(err) {
		return
	}

	var sequrityGroup *string
	for _, output := range cfResp.Stacks[0].Outputs {
		if *output.OutputKey == "NodeSecurityGroup" {
			sequrityGroup = output.OutputValue
			break
		}
	}

	ec2Resp, err := ec2Client.DescribeNetworkInterfaces(&ec2.DescribeNetworkInterfacesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("group-id"),
				Values: []*string{sequrityGroup},
			},
		},
	})

	if checkDeferError(err) {
		return
	}

	for _, networkInterface := range ec2Resp.NetworkInterfaces {
		_, err := ec2Client.DeleteNetworkInterface(&ec2.DeleteNetworkInterfaceInput{
			NetworkInterfaceId: networkInterface.NetworkInterfaceId,
		})

		checkDeferError(err)
	}
}

func DeleteEksClusterVpc(cfClient *cloudformation.CloudFormation, clusterStackName *string) {
	log.Printf("Deleting Amazon EKS Cluster VPC \"%s\"...\n", *clusterStackName)

	resp, err := cfClient.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: clusterStackName,
	})
	if checkDeferError(err) {
		return
	}

	stackId := resp.Stacks[0].StackId

	_, err = cfClient.DeleteStack(&cloudformation.DeleteStackInput{
		StackName: clusterStackName,
	})
	if checkDeferError(err) {
		return
	}

	for {
		resp, err := cfClient.DescribeStacks(&cloudformation.DescribeStacksInput{
			StackName: stackId,
		})
		if checkDeferError(err) {
			if err.(awserr.Error).Code() == "ValidationError" {
				log.Printf("Cluster VPC \"%s\" successfully deleted!\n", *clusterStackName)
			}
			return
		}

		switch *resp.Stacks[0].StackStatus {
		case "DELETE_COMPLETE":
			log.Printf("Cluster VPC \"%s\" successfully deleted!\n", *clusterStackName)
			return
		case "DELETE_IN_PROGRESS":
			time.Sleep(requestInterval)
		default:
			log.Printf("Error: Unexpected stack status: %s\n", *resp.Stacks[0].StackStatus)
			deferError = true
			return
		}
	}
}

func DeleteEksCluster(eksClient *eks.EKS, clusterName *string) {
	log.Printf("Deleting Amazon EKS Cluster \"%s\"...\n", *clusterName)

	_, err := eksClient.DeleteCluster(&eks.DeleteClusterInput{
		Name: clusterName,
	})
	if checkDeferError(err) {
		return
	}

	for {
		resp, err := eksClient.DescribeCluster(&eks.DescribeClusterInput{
			Name: clusterName,
		})

		if err != nil && err.(awserr.Error).Code() == "ResourceNotFoundException" {
			log.Printf("EKS Cluster \"%s\" successfully Deleted!\n", *clusterName)
			return
		}

		if checkDeferError(err) {
			return
		}

		switch *resp.Cluster.Status {
		case "DELETING":
			time.Sleep(requestInterval)
		default:
			log.Printf("Error: Unexpected cluster status: %s\n", *resp.Cluster.Status)
			deferError = true
			return
		}
	}
}

func DeleteEksEc2KeyPair(ec2Client *ec2.EC2, keyPairName *string) {
	log.Printf("Deleting Amazon EC2 key pair \"%s\"...\n", *keyPairName)
	_, err := ec2Client.DeleteKeyPair(&ec2.DeleteKeyPairInput{
		KeyName: keyPairName,
	})
	if checkDeferError(err) {
		return
	}

	log.Printf("Amazon EC2 key pair \"%s\" successfully Deleted!\n", *keyPairName)
}

func DeleteEksWorkerNodes(cfClient *cloudformation.CloudFormation, nodesStackName *string) {
	log.Printf("Deleting Amazon EKS Worker Nodes...\n")

	resp, err := cfClient.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: nodesStackName,
	})
	if checkDeferError(err) {
		return
	}

	stackId := resp.Stacks[0].StackId

	_, err = cfClient.DeleteStack(&cloudformation.DeleteStackInput{
		StackName: nodesStackName,
	})
	if checkDeferError(err) {
		return
	}

	for {
		resp, err := cfClient.DescribeStacks(&cloudformation.DescribeStacksInput{
			StackName: stackId,
		})
		if checkDeferError(err) {
			return
		}

		switch *resp.Stacks[0].StackStatus {
		case "DELETE_COMPLETE":
			log.Printf("EKS Worker Nodes \"%s\" successfully deleted!\n", *nodesStackName)
			return
		case "DELETE_IN_PROGRESS":
			time.Sleep(requestInterval)
		default:
			log.Printf("Error: Unexpected stack status: %s\n", *resp.Stacks[0].StackStatus)
			deferError = true
			return
		}
	}
}

func deleteAWSKubernetesCluster(serviceSuffix string) {
	sess := session.Must(session.NewSession())
	iamClient := iam.New(sess)
	eksClient := eks.New(sess)
	cfClient := cloudformation.New(sess)
	ec2Client := ec2.New(sess)

	// Deleting Amazon EKS Cluster
	clusterName := "nsm" + serviceSuffix
	DeleteEksCluster(eksClient, &clusterName)

	nodesStackName := "nsm-nodes" + serviceSuffix
	// Deleting EC2 Network Interfaces to allow instance to be properly released
	DeleteEC2NetworkInterfaces(ec2Client, cfClient, &nodesStackName)

	// Deleting Amazon EKS Worker Nodes
	DeleteEksWorkerNodes(cfClient, &nodesStackName)

	// Deleting Amazon EKS Cluster VPC
	clusterStackName := "nsm-srv" + serviceSuffix
	DeleteEksClusterVpc(cfClient, &clusterStackName)

	// Deleting Amazon Roles and Keys
	eksRoleName := "nsm-role" + serviceSuffix
	DeleteEksRole(iamClient, &eksRoleName)
	keyPairName := "nsm-key-pair" + serviceSuffix
	DeleteEksEc2KeyPair(ec2Client, &keyPairName)

	if deferError {
		os.Exit(1)
	}
}

func deleteAllKubernetesClusters(saveDuration time.Duration) {
	sess := session.Must(session.NewSession())
	cfClient := cloudformation.New(sess)

	var filter = []*string{aws.String("CREATE_IN_PROGRESS"), aws.String("CREATE_FAILED"), aws.String("CREATE_COMPLETE"), aws.String("ROLLBACK_IN_PROGRESS"), aws.String("ROLLBACK_FAILED"), aws.String("ROLLBACK_COMPLETE"), aws.String("DELETE_IN_PROGRESS"), aws.String("DELETE_FAILED"), aws.String("UPDATE_IN_PROGRESS"), aws.String("UPDATE_COMPLETE_CLEANUP_IN_PROGRESS"), aws.String("UPDATE_COMPLETE"), aws.String("UPDATE_ROLLBACK_IN_PROGRESS"), aws.String("UPDATE_ROLLBACK_FAILED"), aws.String("UPDATE_ROLLBACK_COMPLETE_CLEANUP_IN_PROGRESS"), aws.String("UPDATE_ROLLBACK_COMPLETE"), aws.String("REVIEW_IN_PROGRESS")}
	stacks, err := cfClient.ListStacks(&cloudformation.ListStacksInput{StackStatusFilter:filter})
	if err != nil {
		logrus.Infof("AWS EKS Error: %v", err)
		return
	}

	var wg sync.WaitGroup
	i := 0
	for _, stack := range stacks.StackSummaries {
		stackName := aws.StringValue(stack.StackName)
		if stackName[:7] != "nsm-srv" {
			continue
		}

		if aws.StringValue(stack.StackStatus) == "DELETE_COMPLETE" {
			continue
		}

		if stack.CreationTime.After(time.Now().Add(-1 * saveDuration)) {
			logrus.Infof("Skip cluster: %s (%v)", stackName, stack.CreationTime)
			continue
		}

		wg.Add(1)

		go func() {
			defer wg.Done()
			logrus.Infof("Deleting %s", stackName[7:])
			deleteAWSKubernetesCluster(stackName[7:])
		}()

		i++
		if i%5 == 0 {
			wg.Wait() // Guard from AWS Throttling error
		}
	}
	wg.Wait()
}
