package main

import (
	"log"
	"regexp"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/iam"
)

//nolint:gochecknoglobals
var deferError error

func (ac *AWSCluster) checkDeferError(err error) bool {
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case "NoSuchEntity":
			case "ResourceNotFoundException":
			case "ValidationError":
			case "InvalidParameterValue":
			default:
				log.Printf("Error (%s): %s\n", aerr.Code(), aerr.Message())
				deferError = aerr
				return true
			}
			log.Printf("Warning (%s): %s\n", aerr.Code(), aerr.Message())
		} else {
			log.Printf("Error: %s\n", err.Error())
			deferError = aerr
		}
		return true
	}
	return false
}

func (ac *AWSCluster) deleteEksRole(iamClient *iam.IAM, eksRoleName *string) {
	log.Printf("Deleting EKS service role \"%s\"...\n", *eksRoleName)

	policyArn := "arn:aws:iam::aws:policy/AmazonEKSClusterPolicy"
	_, err := iamClient.DetachRolePolicy(&iam.DetachRolePolicyInput{
		RoleName:  eksRoleName,
		PolicyArn: &policyArn,
	})
	ac.checkDeferError(err)

	policyArn = "arn:aws:iam::aws:policy/AmazonEKSServicePolicy"
	_, err = iamClient.DetachRolePolicy(&iam.DetachRolePolicyInput{
		RoleName:  eksRoleName,
		PolicyArn: &policyArn,
	})
	ac.checkDeferError(err)

	_, err = iamClient.DeleteRole(&iam.DeleteRoleInput{
		RoleName: eksRoleName,
	})
	if ac.checkDeferError(err) {
		return
	}

	log.Printf("Role \"%s\" successfully deleted!\n", *eksRoleName)
}

func (ac *AWSCluster) deleteEC2NetworkInterfaces(ec2Client *ec2.EC2, cfClient *cloudformation.CloudFormation, nodesStackName *string) {
	cfResp, err := cfClient.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: nodesStackName,
	})
	if ac.checkDeferError(err) {
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

	if ac.checkDeferError(err) {
		return
	}

	for _, networkInterface := range ec2Resp.NetworkInterfaces {
		_, err := ec2Client.DeleteNetworkInterface(&ec2.DeleteNetworkInterfaceInput{
			NetworkInterfaceId: networkInterface.NetworkInterfaceId,
		})

		ac.checkDeferError(err)
	}
}

//nolint:funlen
func (ac *AWSCluster) deleteEksClusterVpc(cfClient *cloudformation.CloudFormation, ec2Client *ec2.EC2, clusterStackName *string) {
	log.Printf("Deleting Amazon EKS Cluster VPC \"%s\"...\n", *clusterStackName)

	resp, err := cfClient.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: clusterStackName,
	})
	if ac.checkDeferError(err) {
		return
	}
	outputsMap := outputsToMap(resp.Stacks[0].Outputs)

	output, err := ec2Client.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []*string{outputsMap.VpcId},
			},
		},
	})
	if !ac.checkDeferError(err) {
		for _, sg := range output.SecurityGroups {
			if *sg.GroupName != "default" {
				log.Printf("Deleting EC2 Security Group %s...", *sg.GroupId)
				_, err = ec2Client.DeleteSecurityGroup(&ec2.DeleteSecurityGroupInput{GroupId: sg.GroupId})
				ac.checkDeferError(err)
			}
		}
	}

	_, err = ec2Client.DeleteVpc(&ec2.DeleteVpcInput{VpcId: outputsMap.VpcId})
	ac.checkDeferError(err)

	stackId := resp.Stacks[0].StackId

	_, err = cfClient.DeleteStack(&cloudformation.DeleteStackInput{
		StackName: clusterStackName,
	})
	if ac.checkDeferError(err) {
		return
	}

	for {
		resp, err := cfClient.DescribeStacks(&cloudformation.DescribeStacksInput{
			StackName: stackId,
		})
		if ac.checkDeferError(err) {
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
			deferError = errors.Errorf("unexpected stack status: %s", *resp.Stacks[0].StackStatus)
			return
		}
	}
}

func (ac *AWSCluster) deleteEksCluster(eksClient *eks.EKS, clusterName *string) {
	log.Printf("Deleting Amazon EKS Cluster \"%s\"...\n", *clusterName)

	_, err := eksClient.DeleteCluster(&eks.DeleteClusterInput{
		Name: clusterName,
	})
	if ac.checkDeferError(err) {
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

		if ac.checkDeferError(err) {
			return
		}

		switch *resp.Cluster.Status {
		case "DELETING":
			time.Sleep(requestInterval)
		default:
			log.Printf("Error: Unexpected cluster status: %s\n", *resp.Cluster.Status)
			deferError = errors.Errorf("unexpected cluster status: %s", *resp.Cluster.Status)
			return
		}
	}
}

func (ac *AWSCluster) deleteEksEc2KeyPair(ec2Client *ec2.EC2, keyPairName *string) {
	log.Printf("Deleting Amazon EC2 key pair \"%s\"...\n", *keyPairName)
	_, err := ec2Client.DeleteKeyPair(&ec2.DeleteKeyPairInput{
		KeyName: keyPairName,
	})
	if ac.checkDeferError(err) {
		return
	}

	log.Printf("Amazon EC2 key pair \"%s\" successfully Deleted!\n", *keyPairName)
}

func (ac *AWSCluster) deleteEksWorkerNodes(cfClient *cloudformation.CloudFormation, nodesStackName *string) {
	log.Printf("Deleting Amazon EKS Worker Nodes...\n")

	resp, err := cfClient.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: nodesStackName,
	})
	if ac.checkDeferError(err) {
		return
	}

	stackId := resp.Stacks[0].StackId

	_, err = cfClient.DeleteStack(&cloudformation.DeleteStackInput{
		StackName: nodesStackName,
	})
	if ac.checkDeferError(err) {
		return
	}

	for {
		resp, err := cfClient.DescribeStacks(&cloudformation.DescribeStacksInput{
			StackName: stackId,
		})
		if ac.checkDeferError(err) {
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
			deferError = errors.Errorf("unexpected stack status: %s", *resp.Stacks[0].StackStatus)
			return
		}
	}
}

// DeleteAWSKubernetesCluster - deleting AWS cluster instances
func (ac *AWSCluster) DeleteAWSKubernetesCluster() error {
	sess := session.Must(session.NewSession())
	iamClient := iam.New(sess)
	eksClient := eks.New(sess)
	cfClient := cloudformation.New(sess)
	ec2Client := ec2.New(sess)

	// Deleting Amazon EKS Cluster
	clusterName := awsClusterPrefix + ac.serviceSuffix
	ac.deleteEksCluster(eksClient, &clusterName)

	nodesStackName := awsNodesStackPrefix + ac.serviceSuffix
	// Deleting EC2 Network Interfaces to allow instance to be properly released
	ac.deleteEC2NetworkInterfaces(ec2Client, cfClient, &nodesStackName)

	// Deleting Amazon EKS Worker Nodes
	ac.deleteEksWorkerNodes(cfClient, &nodesStackName)

	// Deleting Amazon EKS Cluster VPC
	clusterStackName := awsClusterStackPrefix + ac.serviceSuffix
	ac.deleteEksClusterVpc(cfClient, ec2Client, &clusterStackName)

	// Deleting Amazon Roles and Keys
	eksRoleName := awsRolePrefix + ac.serviceSuffix
	ac.deleteEksRole(iamClient, &eksRoleName)
	keyPairName := awsKeyPairPrefix + ac.serviceSuffix
	ac.deleteEksEc2KeyPair(ec2Client, &keyPairName)

	return deferError
}

// DeleteAllKubernetesClusters - removes all aws clusters older than saveDuration and filtered by name
func DeleteAllKubernetesClusters(saveDuration time.Duration, namePattern string) {
	sess := session.Must(session.NewSession())
	cfClient := cloudformation.New(sess)

	var filter = []*string{aws.String("CREATE_IN_PROGRESS"), aws.String("CREATE_FAILED"), aws.String("CREATE_COMPLETE"), aws.String("ROLLBACK_IN_PROGRESS"), aws.String("ROLLBACK_FAILED"), aws.String("ROLLBACK_COMPLETE"), aws.String("DELETE_IN_PROGRESS"), aws.String("DELETE_FAILED"), aws.String("UPDATE_IN_PROGRESS"), aws.String("UPDATE_COMPLETE_CLEANUP_IN_PROGRESS"), aws.String("UPDATE_COMPLETE"), aws.String("UPDATE_ROLLBACK_IN_PROGRESS"), aws.String("UPDATE_ROLLBACK_FAILED"), aws.String("UPDATE_ROLLBACK_COMPLETE_CLEANUP_IN_PROGRESS"), aws.String("UPDATE_ROLLBACK_COMPLETE"), aws.String("REVIEW_IN_PROGRESS")}
	stacks, err := cfClient.ListStacks(&cloudformation.ListStacksInput{StackStatusFilter: filter})
	if err != nil {
		logrus.Infof("AWS EKS Error: %v", err)
		return
	}

	var wg sync.WaitGroup
	i := 0
	for _, stack := range stacks.StackSummaries {
		stackName := aws.StringValue(stack.StackName)
		if stackName[:len(awsClusterStackPrefix)] != "nsm-srv" {
			continue
		}

		if aws.StringValue(stack.StackStatus) == "DELETE_COMPLETE" {
			continue
		}

		if matched, err := regexp.MatchString(namePattern, stackName[len(awsClusterStackPrefix):]); err != nil || !matched {
			logrus.Infof("Skip cluster: %s (matched: %v; err: %v)", stackName[len(awsClusterStackPrefix):], matched, err)
			continue
		}

		if stack.CreationTime.After(time.Now().Add(-1 * saveDuration)) {
			logrus.Infof("Skip cluster: %s (created: %v)", stackName[len(awsClusterStackPrefix):], stack.CreationTime)
			continue
		}

		wg.Add(1)

		go func() {
			defer wg.Done()
			for att := 0; att < 3; att++ {
				logrus.Infof("Deleting %s (created %v), attempt %d", stackName[len(awsClusterStackPrefix):], stack.CreationTime, att+1)
				err := NewAWSCluster(stackName[len(awsClusterStackPrefix):]).DeleteAWSKubernetesCluster()
				if err == nil {
					break
				}
			}
		}()

		i++
		if i%5 == 0 {
			wg.Wait() // Guard from AWS Throttling error
		}
	}
	wg.Wait()
}
