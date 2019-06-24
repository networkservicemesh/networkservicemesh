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

	"github.com/aws/aws-sdk-go/aws"
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

func checkError(err error) {
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case "EntityAlreadyExists":
			case "AlreadyExistsException":
			case "ResourceInUseException":
			case "InvalidKeyPair.Duplicate":
			case "InvalidPermission.Duplicate":
			case "Throttling":
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
		StackName:       clusterStackName,
		TemplateBody:    &s,
		DisableRollback: aws.Bool(true),
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
			time.Sleep(requestInterval)
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
		Version: aws.String("1.13"),
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
			time.Sleep(requestInterval)
		default:
			log.Fatalf("Error: Unexpected cluster status: %s\n", *resp.Cluster.Status)
		}
	}
}

func CreateKubeConfigFile(cluster *eks.Cluster) {
	kubeconfigFile := os.Getenv("KUBECONFIG")
	if len(kubeconfigFile) == 0 {
		kubeconfigFile = path.Join(os.Getenv("HOME"), ".kube/config")
	}
	kc, err := ioutil.ReadFile(path.Join(currentPath, "kube-config-template"))
	checkError(err)
	kubeconfig := string(kc)

	kubeconfig = strings.Replace(kubeconfig, "<CERT>", *cluster.CertificateAuthority.Data, -1)
	kubeconfig = strings.Replace(kubeconfig, "<SERVER_ENDPOINT>", *cluster.Endpoint, -1)
	kubeconfig = strings.Replace(kubeconfig, "<SERVER_NAME>", *cluster.Arn, -1)
	kubeconfig = strings.Replace(kubeconfig, "<CLUSTER_NAME>", *cluster.Name, -1)

	err = os.Mkdir(path.Dir(kubeconfigFile), 0775)
	err = ioutil.WriteFile(kubeconfigFile, []byte(kubeconfig), 0644)
	checkError(err)

	log.Printf("Updated context %s in %s\n", *cluster.Arn, kubeconfigFile)
}

func CreateEksEc2KeyPair(ec2Client *ec2.EC2, keyPairName *string) {
	log.Printf("Creating Amazon EC2 key pair \"%s\"...\n", *keyPairName)
	var resp *ec2.CreateKeyPairOutput
	resp, err := ec2Client.CreateKeyPair(&ec2.CreateKeyPairInput{
		KeyName: keyPairName,
	})
	if err != nil && err.(awserr.Error).Code() == "InvalidKeyPair.Duplicate" {
		return
	}
	checkError(err)

	keyFile := "nsm-key-pair" + os.Getenv("NSM_AWS_SERVICE_SUFFIX")
	os.Remove(keyFile)
	err = ioutil.WriteFile(keyFile, []byte(*resp.KeyMaterial), 0400)

	checkError(err)
	log.Printf("Amazon EC2 key pair \"%s\" successfully created!\n", *keyPairName)
}

func createEksWorkerNodes(cfClient *cloudformation.CloudFormation, nodesStackName *string, nodeGroupName *string, clusterName *string, keyPairName *string, clusterStackOutputs *OutputsMap) (*string, *string) {
	log.Printf("Creating Amazon EKS Worker Nodes on cluster \"%s\"...\n", *clusterName)

	sf, err := ioutil.ReadFile(path.Join(currentPath, "amazon-eks-nodegroup.yaml"))
	checkError(err)

	s := string(sf)

	// Base image for Amazon EKS worker nodes
	// with Kubernetes version 1.13.7
	// for region us-east-2.
	// Amazon EKS-Optimized AMI list: https://docs.aws.amazon.com/eks/latest/userguide/eks-optimized-ami.html
	eksAmi := aws.String("ami-0485258c2d1c3608f")

	_, err = cfClient.CreateStack(&cloudformation.CreateStackInput{
		StackName:       nodesStackName,
		TemplateBody:    &s,
		DisableRollback: aws.Bool(true),
		Capabilities:    []*string{aws.String("CAPABILITY_IAM")},
		Parameters: []*cloudformation.Parameter{
			{
				ParameterKey:   aws.String("KeyName"),
				ParameterValue: keyPairName,
			},
			{
				ParameterKey:   aws.String("NodeImageId"),
				ParameterValue: eksAmi,
			},
			{
				ParameterKey:   aws.String("ClusterName"),
				ParameterValue: clusterName,
			},
			{
				ParameterKey:   aws.String("NodeGroupName"),
				ParameterValue: nodeGroupName,
			},
			{
				ParameterKey:   aws.String("ClusterControlPlaneSecurityGroup"),
				ParameterValue: clusterStackOutputs.SecurityGroups,
			},
			{
				ParameterKey:   aws.String("VpcId"),
				ParameterValue: clusterStackOutputs.VpcId,
			},
			{
				ParameterKey:   aws.String("Subnets"),
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

			var sequrityGroup *string
			var instanceRole *string

			for _, output := range resp.Stacks[0].Outputs {
				if *output.OutputKey == "NodeSecurityGroup" {
					sequrityGroup = output.OutputValue
					break
				}
			}

			for _, output := range resp.Stacks[0].Outputs {
				if *output.OutputKey == "NodeInstanceRole" {
					instanceRole = output.OutputValue
				}
			}
			return sequrityGroup, instanceRole
		case "CREATE_IN_PROGRESS":
			time.Sleep(requestInterval)
		default:
			log.Fatalf("Error: Unexpected stack status: %s\n", *resp.Stacks[0].StackStatus)
		}
	}
}

func AuthorizeSecurityGroupIngress(ec2client *ec2.EC2, groupId *string) {
	_, err := ec2client.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: groupId,
		IpPermissions: []*ec2.IpPermission{
			{
				IpProtocol: aws.String("tcp"),
				ToPort:     aws.Int64(65535),
				FromPort:   aws.Int64(0),
				IpRanges: []*ec2.IpRange{
					{
						CidrIp:      aws.String("0.0.0.0/0"),
						Description: aws.String("Remote ip4 access"),
					},
				},
				Ipv6Ranges: []*ec2.Ipv6Range{
					{
						CidrIpv6:    aws.String("::/0"),
						Description: aws.String("Remote ip6 access"),
					},
				},
			},
			{
				IpProtocol: aws.String("udp"),
				ToPort:     aws.Int64(65535),
				FromPort:   aws.Int64(0),
				IpRanges: []*ec2.IpRange{
					{
						CidrIp:      aws.String("0.0.0.0/0"),
						Description: aws.String("Remote ip4 access"),
					},
				},
				Ipv6Ranges: []*ec2.Ipv6Range{
					{
						CidrIpv6:    aws.String("::/0"),
						Description: aws.String("Remote ip6 access"),
					},
				},
			},
			{
				IpProtocol: aws.String("tcp"),
				ToPort:     aws.Int64(80),
				FromPort:   aws.Int64(80),
				IpRanges: []*ec2.IpRange{
					{
						CidrIp:      aws.String("0.0.0.0/0"),
						Description: aws.String("Remote ip4 access"),
					},
				},
				Ipv6Ranges: []*ec2.Ipv6Range{
					{
						CidrIpv6:    aws.String("::/0"),
						Description: aws.String("Remote ip6 access"),
					},
				},
			},
			{
				IpProtocol: aws.String("tcp"),
				ToPort:     aws.Int64(5100),
				FromPort:   aws.Int64(5000),
				IpRanges: []*ec2.IpRange{
					{
						CidrIp:      aws.String("0.0.0.0/0"),
						Description: aws.String("Remote ip4 access"),
					},
				},
				Ipv6Ranges: []*ec2.Ipv6Range{
					{
						CidrIpv6:    aws.String("::/0"),
						Description: aws.String("Remote ip6 access"),
					},
				},
			},
			{
				IpProtocol: aws.String("udp"),
				ToPort:     aws.Int64(4789),
				FromPort:   aws.Int64(4789),
				IpRanges: []*ec2.IpRange{
					{
						CidrIp:      aws.String("0.0.0.0/0"),
						Description: aws.String("Remote ip4 access"),
					},
				},
				Ipv6Ranges: []*ec2.Ipv6Range{
					{
						CidrIpv6:    aws.String("::/0"),
						Description: aws.String("Remote ip6 access"),
					},
				},
			},
		},
	})
	checkError(err)
}

func CreateSSHConfig(ec2client *ec2.EC2, vpcId *string, scpConfigFileName string) {
	res, err := ec2client.DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("vpc-id"),
				Values: []*string{
					vpcId,
				},
			},
		},
	})
	checkError(err)

	scpConfigBytes, err := ioutil.ReadFile(path.Join(currentPath, "scp-config-template"))
	checkError(err)
	scpConfig := string(scpConfigBytes)

	i := 0
	hostNames := [2]string{"<EC2MASTER>", "<EC2WORKER>"}
	for _, reserv := range res.Reservations {
		for _, v := range reserv.Instances {
			if len(hostNames) > i {
				scpConfig = strings.Replace(scpConfig, hostNames[i], *v.PublicIpAddress, 1)
				i++
			}
		}
	}

	err = ioutil.WriteFile(path.Join(currentPath, scpConfigFileName), []byte(scpConfig), 0666)
	checkError(err)
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
	nodeSequrityGroup, nodeInstanceRole := createEksWorkerNodes(cfClient, &nodesStackName, &nodeGroupName, &clusterName, &keyPairName, clusterStackOutputs)

	AuthorizeSecurityGroupIngress(ec2Client, nodeSequrityGroup)

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

	CreateSSHConfig(ec2Client, clusterStackOutputs.VpcId, "scp-config"+serviceSuffix)
}
