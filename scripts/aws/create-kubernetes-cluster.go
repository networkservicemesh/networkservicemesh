package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"time"
)

import (
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/iam"
)

type OutputsMap struct {
	SecurityGroups *string
	VpcId          *string
	SubnetIds      *string
}

var _, currentFilePath, _, _ = runtime.Caller(0)
var currentPath = path.Dir(currentFilePath)

func strp(str string) *string {
	return &str
}

func checkError(err error) {
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case "EntityAlreadyExists":
			case "AlreadyExistsException":
			case "ResourceInUseException":
			case "InvalidKeyPair.Duplicate":
			default:
				log.Fatalf("Error (%s): %s\n", aerr.Code(), aerr.Message())
			}
			log.Printf("Warning (%s): %s\n", aerr.Code(), aerr.Message())
		} else {
			log.Fatalf("Error: %s\n", err.Error())
		}
	}
}

func OutputsToMap(outputs []*cloudformation.Output) *OutputsMap {
	res := &OutputsMap{}
	for _, v := range outputs {
		switch *v.OutputKey {
		case "SecurityGroups":
			res.SecurityGroups = v.OutputValue
		case "VpcId":
			res.VpcId = v.OutputValue
		case "SubnetIds":
			res.SubnetIds = v.OutputValue
		}
	}
	return res
}

func CreateEksRole(iamClient *iam.IAM, eksRoleName *string) *string {
	log.Printf("Creating EKS service role \"%s\"...\n", *eksRoleName)
	roleDescription := "Allows EKS to manage clusters on your behalf."

	rpf, err := ioutil.ReadFile(path.Join(currentPath, "amazon-eks-role-policy.json"))
	checkError(err)

	rps := string(rpf)
	_, err = iamClient.CreateRole(&iam.CreateRoleInput{
		RoleName:                 eksRoleName,
		Description:              &roleDescription,
		AssumeRolePolicyDocument: &rps,
	})
	checkError(err)

	policyArn := "arn:aws:iam::aws:policy/AmazonEKSClusterPolicy"
	_, err = iamClient.AttachRolePolicy(&iam.AttachRolePolicyInput{
		RoleName:  eksRoleName,
		PolicyArn: &policyArn,
	})
	checkError(err)

	policyArn = "arn:aws:iam::aws:policy/AmazonEKSServicePolicy"
	_, err = iamClient.AttachRolePolicy(&iam.AttachRolePolicyInput{
		RoleName:  eksRoleName,
		PolicyArn: &policyArn,
	})
	checkError(err)

	result, err := iamClient.GetRole(&iam.GetRoleInput{
		RoleName: eksRoleName,
	})
	checkError(err)

	log.Printf("Role \"%s\"(%s) successfully created!\n", *eksRoleName, *result.Role.Arn)

	return result.Role.Arn
}

func CreateEksClusterVpc(cfClient *cloudformation.CloudFormation, clusterStackName *string) *OutputsMap {
	log.Printf("Creating Amazon EKS Cluster VPC \"%s\"...\n", *clusterStackName)

	sf, err := ioutil.ReadFile(path.Join(currentPath, "amazon-eks-vpc.yaml"))
	checkError(err)

	s := string(sf)
	_, err = cfClient.CreateStack(&cloudformation.CreateStackInput{
		StackName:    clusterStackName,
		TemplateBody: &s,
	})
	checkError(err)

	for {
		resp, err := cfClient.DescribeStacks(&cloudformation.DescribeStacksInput{
			StackName: clusterStackName,
		})
		checkError(err)

		switch *resp.Stacks[0].StackStatus {
		case "CREATE_COMPLETE":
			log.Printf("Cluster VPC \"%s\" successfully created!\n", *clusterStackName)
			return OutputsToMap(resp.Stacks[0].Outputs)
		case "CREATE_IN_PROGRESS":
			time.Sleep(time.Second)
		default:
			log.Fatalf("Error: Unexpected stack status: %s\n", *resp.Stacks[0].StackStatus)
		}
	}
}

func CreateEksCluster(eksClient *eks.EKS, clusterName *string, eksRoleArn *string, clusterStackOutputs *OutputsMap) *eks.Cluster {
	log.Printf("Creating Amazon EKS Cluster \"%s\"...\n", *clusterName)
	subnetIdsTemp := strings.Split(*clusterStackOutputs.SubnetIds, ",")
	var subnetIds []*string
	for i := range subnetIdsTemp {
		subnetIds = append(subnetIds, &subnetIdsTemp[i])
	}
	endpointPrivateAccess := false
	endpointPublicAccess := true

	_, err := eksClient.CreateCluster(&eks.CreateClusterInput{
		Name:    clusterName,
		RoleArn: eksRoleArn,
		ResourcesVpcConfig: &eks.VpcConfigRequest{
			SubnetIds: subnetIds,
			SecurityGroupIds: []*string{
				clusterStackOutputs.SecurityGroups,
			},
			EndpointPrivateAccess: &endpointPrivateAccess,
			EndpointPublicAccess:  &endpointPublicAccess,
		},
	})
	checkError(err)

	for {
		resp, err := eksClient.DescribeCluster(&eks.DescribeClusterInput{
			Name: clusterName,
		})
		checkError(err)

		switch *resp.Cluster.Status {
		case "ACTIVE":
			log.Printf("EKS Cluster \"%s\" successfully created!\n", *clusterName)
			return resp.Cluster
		case "CREATING":
			time.Sleep(time.Second)
		default:
			log.Fatalf("Error: Unexpected cluster status: %s\n", *resp.Cluster.Status)
		}
	}
}

func CreateKubeConfigFile(cluster *eks.Cluster) {
	kubeconfigFile := os.Getenv("KUBECONFIG")
	if len(kubeconfigFile) == 0 {
		kubeconfigFile = os.Getenv("HOME") + "/.kube/config"
	}
	kc, err := ioutil.ReadFile(path.Join(currentPath, "kube-config-template"))
	checkError(err)
	kubeconfig := string(kc)

	kubeconfig = strings.Replace(kubeconfig, "<CERT>", *cluster.CertificateAuthority.Data, -1)
	kubeconfig = strings.Replace(kubeconfig, "<SERVER_ENDPOINT>", *cluster.Endpoint, -1)
	kubeconfig = strings.Replace(kubeconfig, "<SERVER_NAME>", *cluster.Arn, -1)

	err = ioutil.WriteFile(kubeconfigFile, []byte(kubeconfig), 0644)
	checkError(err)

	log.Printf("Updated context %s in %s\n", *cluster.Arn, kubeconfigFile)
}

func CreateEksEc2KeyPair(ec2Client *ec2.EC2, keyPairName *string) {
	log.Printf("Creating Amazon EC2 key pair \"%s\"...\n", *keyPairName)
	_, err := ec2Client.CreateKeyPair(&ec2.CreateKeyPairInput{
		KeyName: keyPairName,
	})
	checkError(err)

	log.Printf("Amazon EC2 key pair \"%s\" successfully created!\n", *keyPairName)
}

func createEksWorkerNodes(cfClient *cloudformation.CloudFormation, nodesStackName *string, nodeGroupName *string, clusterName *string, keyPairName *string, clusterStackOutputs *OutputsMap) *string {
	log.Printf("Creating Amazon EKS Worker Nodes on cluster \"%s\"...\n", *clusterName)

	sf, err := ioutil.ReadFile(path.Join(currentPath, "amazon-eks-nodegroup.yaml"))
	checkError(err)

	s := string(sf)

	_, err = cfClient.CreateStack(&cloudformation.CreateStackInput{
		StackName:    nodesStackName,
		TemplateBody: &s,
		Capabilities: []*string{strp("CAPABILITY_IAM")},
		Parameters: []*cloudformation.Parameter{
			{
				ParameterKey:   strp("KeyName"),
				ParameterValue: keyPairName,
			},
			{
				ParameterKey:   strp("NodeImageId"),
				ParameterValue: strp("ami-0484545fe7d3da96f"),
			},
			{
				ParameterKey:   strp("ClusterName"),
				ParameterValue: clusterName,
			},
			{
				ParameterKey:   strp("NodeGroupName"),
				ParameterValue: nodeGroupName,
			},
			{
				ParameterKey:   strp("ClusterControlPlaneSecurityGroup"),
				ParameterValue: clusterStackOutputs.SecurityGroups,
			},
			{
				ParameterKey:   strp("VpcId"),
				ParameterValue: clusterStackOutputs.VpcId,
			},
			{
				ParameterKey:   strp("Subnets"),
				ParameterValue: clusterStackOutputs.SubnetIds,
			},
		},
	})
	checkError(err)

	for {
		resp, err := cfClient.DescribeStacks(&cloudformation.DescribeStacksInput{
			StackName: nodesStackName,
		})
		checkError(err)

		switch *resp.Stacks[0].StackStatus {
		case "CREATE_COMPLETE":
			log.Printf("EKS Worker Nodes \"%s\" successfully created!\n", *nodesStackName)
			for _, output := range resp.Stacks[0].Outputs {
				if *output.OutputKey == "NodeInstanceRole" {
					return output.OutputValue
				}
			}
			return nil
		case "CREATE_IN_PROGRESS":
			time.Sleep(time.Second)
		default:
			log.Fatalf("Error: Unexpected stack status: %s\n", *resp.Stacks[0].StackStatus)
		}
	}
}

func createAWSKubernetesCluster() {
	sess := session.Must(session.NewSession())
	iamClient := iam.New(sess)
	eksClient := eks.New(sess)
	cfClient := cloudformation.New(sess)
	ec2Client := ec2.New(sess)

	// Creating Amazon EKS Role
	serviceSuffix := os.Getenv("NSM_AWS_SERVICE_SUFFIX")
	eksRoleName := "nsm-role" + serviceSuffix
	eksRoleArn := CreateEksRole(iamClient, &eksRoleName)

	// Creating Amazon EKS Cluster VPC
	clusterStackName := "nsm-srv" + serviceSuffix
	clusterStackOutputs := CreateEksClusterVpc(cfClient, &clusterStackName)

	// Creating Amazon EKS Cluster
	clusterName := "nsm" + serviceSuffix
	cluster := CreateEksCluster(eksClient, &clusterName, eksRoleArn, clusterStackOutputs)

	// Creating kubeconfig file
	CreateKubeConfigFile(cluster)

	// Creating Amazon EKS Worker Nodes
	keyPairName := "nsm-key-pair" + serviceSuffix
	CreateEksEc2KeyPair(ec2Client, &keyPairName)

	nodesStackName := "nsm-nodes" + serviceSuffix
	nodeGroupName := "nsm-node-group" + serviceSuffix
	nodeInstanceRole := createEksWorkerNodes(cfClient, &nodesStackName, &nodeGroupName, &clusterName, &keyPairName, clusterStackOutputs)

	// Enable worker nodes to join the cluster
	sf, err := ioutil.ReadFile(path.Join(currentPath, "aws-auth-cm-temp.yaml"))
	checkError(err)

	f, err := ioutil.TempFile(os.TempDir(), "aws-auth-cm-temp-*.yaml")
	checkError(err)

	s := string(sf)
	_, err = f.Write([]byte(strings.Replace(s, "<NodeInstanceRole>", *nodeInstanceRole, -1)))
	checkError(err)

	_ = f.Close()

	log.Printf("> kubectl %s %s %s", "apply", "-f", f.Name())
	cmd := exec.Command("kubectl", "apply", "-f", f.Name())
	var out bytes.Buffer
	cmd.Stdout = &out
	err = cmd.Run()
	checkError(err)
	log.Printf(out.String())
	_ = os.Remove(f.Name())
}
