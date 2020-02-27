package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
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

func (ac *AWSCluster) checkError(err error) {
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case "EntityAlreadyExists":
			case "AlreadyExistsException":
			case "ResourceInUseException":
			case "InvalidKeyPair.Duplicate":
			case "InvalidPermission.Duplicate":
			default:
				panic(aerr)
			}
			log.Printf("Warning (%s): %s\n", aerr.Code(), aerr.Message())
		} else {
			panic(err)
		}
	}
}

func outputsToMap(outputs []*cloudformation.Output) *OutputsMap {
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

func (ac *AWSCluster) createEksRole(iamClient *iam.IAM, eksRoleName *string) *string {
	log.Printf("Creating EKS service role \"%s\"...\n", *eksRoleName)
	roleDescription := "Allows EKS to manage clusters on your behalf."

	rpf, err := ioutil.ReadFile(path.Join(ac.configPath, "amazon-eks-role-policy.json"))
	ac.checkError(err)

	rps := string(rpf)
	_, err = iamClient.CreateRole(&iam.CreateRoleInput{
		RoleName:                 eksRoleName,
		Description:              &roleDescription,
		AssumeRolePolicyDocument: &rps,
	})
	ac.checkError(err)

	policyArn := "arn:aws:iam::aws:policy/AmazonEKSClusterPolicy"
	_, err = iamClient.AttachRolePolicy(&iam.AttachRolePolicyInput{
		RoleName:  eksRoleName,
		PolicyArn: &policyArn,
	})
	ac.checkError(err)

	policyArn = "arn:aws:iam::aws:policy/AmazonEKSServicePolicy"
	_, err = iamClient.AttachRolePolicy(&iam.AttachRolePolicyInput{
		RoleName:  eksRoleName,
		PolicyArn: &policyArn,
	})
	ac.checkError(err)

	result, err := iamClient.GetRole(&iam.GetRoleInput{
		RoleName: eksRoleName,
	})
	ac.checkError(err)

	log.Printf("Role \"%s\"(%s) successfully created!\n", *eksRoleName, *result.Role.Arn)

	return result.Role.Arn
}

func (ac *AWSCluster) createEksClusterVpc(cfClient *cloudformation.CloudFormation, clusterStackName *string) *OutputsMap {
	log.Printf("Creating Amazon EKS Cluster VPC \"%s\"...\n", *clusterStackName)

	sf, err := ioutil.ReadFile(path.Join(ac.configPath, "amazon-eks-vpc.yaml"))
	ac.checkError(err)

	s := string(sf)
	_, err = cfClient.CreateStack(&cloudformation.CreateStackInput{
		StackName:       clusterStackName,
		TemplateBody:    &s,
		DisableRollback: aws.Bool(true),
	})
	ac.checkError(err)

	for {
		resp, err := cfClient.DescribeStacks(&cloudformation.DescribeStacksInput{
			StackName: clusterStackName,
		})
		ac.checkError(err)

		switch *resp.Stacks[0].StackStatus {
		case "CREATE_COMPLETE":
			log.Printf("Cluster VPC \"%s\" successfully created!\n", *clusterStackName)
			return outputsToMap(resp.Stacks[0].Outputs)
		case "CREATE_IN_PROGRESS":
			time.Sleep(requestInterval)
		default:
			log.Fatalf("Error: Unexpected stack status: %s\n", *resp.Stacks[0].StackStatus)
		}
	}
}

func (ac *AWSCluster) createEksCluster(eksClient *eks.EKS, clusterName, eksRoleArn *string, clusterStackOutputs *OutputsMap) *eks.Cluster {
	log.Printf("Creating Amazon EKS Cluster \"%s\"...\n", *clusterName)
	subnetIdsTemp := strings.Split(*clusterStackOutputs.SubnetIds, ",")
	var subnetIds []*string
	for i := range subnetIdsTemp {
		subnetIds = append(subnetIds, &subnetIdsTemp[i])
	}

	_, err := eksClient.CreateCluster(&eks.CreateClusterInput{
		Name:    clusterName,
		RoleArn: eksRoleArn,
		ResourcesVpcConfig: &eks.VpcConfigRequest{
			SubnetIds: subnetIds,
			SecurityGroupIds: []*string{
				clusterStackOutputs.SecurityGroups,
			},
			EndpointPrivateAccess: aws.Bool(true),
			EndpointPublicAccess:  aws.Bool(true),
		},
		Version: aws.String("1.14"),
	})
	ac.checkError(err)

	for {
		resp, err := eksClient.DescribeCluster(&eks.DescribeClusterInput{
			Name: clusterName,
		})
		ac.checkError(err)

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

func (ac *AWSCluster) createKubeConfigFile(cluster *eks.Cluster) {
	kubeconfigFile := os.Getenv("KUBECONFIG")
	if len(kubeconfigFile) == 0 {
		kubeconfigFile = path.Join(os.Getenv("HOME"), ".kube/config")
	}
	kc, err := ioutil.ReadFile(path.Join(ac.configPath, "kube-config-template"))
	ac.checkError(err)
	kubeconfig := string(kc)

	kubeconfig = strings.Replace(kubeconfig, "<CERT>", *cluster.CertificateAuthority.Data, -1)
	kubeconfig = strings.Replace(kubeconfig, "<SERVER_ENDPOINT>", *cluster.Endpoint, -1)
	kubeconfig = strings.Replace(kubeconfig, "<SERVER_NAME>", *cluster.Arn, -1)
	kubeconfig = strings.Replace(kubeconfig, "<CLUSTER_NAME>", *cluster.Name, -1)

	err = os.Mkdir(path.Dir(kubeconfigFile), 0775)
	err = ioutil.WriteFile(kubeconfigFile, []byte(kubeconfig), 0644)
	ac.checkError(err)

	log.Printf("Updated context %s in %s\n", *cluster.Arn, kubeconfigFile)
}

func (ac *AWSCluster) createEksEc2KeyPair(ec2Client *ec2.EC2, keyPairName *string) {
	log.Printf("Creating Amazon EC2 key pair \"%s\"...\n", *keyPairName)
	var resp *ec2.CreateKeyPairOutput
	resp, err := ec2Client.CreateKeyPair(&ec2.CreateKeyPairInput{
		KeyName: keyPairName,
	})
	if err != nil && err.(awserr.Error).Code() == "InvalidKeyPair.Duplicate" {
		return
	}
	ac.checkError(err)

	keyFile := "nsm-key-pair" + os.Getenv("NSM_AWS_SERVICE_SUFFIX")
	_ = os.Remove(keyFile)

	err = ioutil.WriteFile(keyFile, []byte(*resp.KeyMaterial), 0400)
	ac.checkError(err)

	log.Printf("Amazon EC2 key pair \"%s\" successfully created!\n", *keyPairName)
}

//nolint:funlen
func (ac *AWSCluster) createEksWorkerNodes(cfClient *cloudformation.CloudFormation, nodesStackName, nodeGroupName, clusterName, keyPairName *string, clusterStackOutputs *OutputsMap) (*string, *string) {
	log.Printf("Creating Amazon EKS Worker Nodes on cluster \"%s\"...\n", *clusterName)

	sf, err := ioutil.ReadFile(path.Join(ac.configPath, "amazon-eks-nodegroup.yaml"))
	ac.checkError(err)

	s := string(sf)

	// Base image for Amazon EKS worker nodes
	// with Kubernetes version 1.14.7
	// for region us-east-2.
	// Amazon EKS-Optimized AMI list: https://docs.aws.amazon.com/eks/latest/userguide/eks-optimized-ami.html
	eksAmi := aws.String("ami-053250833d1030033")

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
				ParameterValue: &strings.Split(*clusterStackOutputs.SubnetIds, ",")[0],
			},
		},
	})
	ac.checkError(err)

	for {
		resp, err := cfClient.DescribeStacks(&cloudformation.DescribeStacksInput{
			StackName: nodesStackName,
		})
		ac.checkError(err)

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

//nolint:funlen
func (ac *AWSCluster) authorizeSecurityGroupIngress(ec2client *ec2.EC2, groupID *string) {
	_, err := ec2client.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: groupID,
		IpPermissions: []*ec2.IpPermission{
			{
				IpProtocol: aws.String("tcp"),
				ToPort:     aws.Int64(22),
				FromPort:   aws.Int64(22),
				IpRanges: []*ec2.IpRange{
					{
						CidrIp:      aws.String("0.0.0.0/0"),
						Description: aws.String("SSH access"),
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
			{
				IpProtocol: aws.String("udp"),
				ToPort:     aws.Int64(52000),
				FromPort:   aws.Int64(51820),
				IpRanges: []*ec2.IpRange{
					{
						CidrIp:      aws.String("0.0.0.0/0"),
						Description: aws.String("Wireguard ip4 access"),
					},
				},
				Ipv6Ranges: []*ec2.Ipv6Range{
					{
						CidrIpv6:    aws.String("::/0"),
						Description: aws.String("Wireguard ip6 access"),
					},
				},
			},
		},
	})
	ac.checkError(err)
}

func (ac *AWSCluster) createSSHConfig(ec2client *ec2.EC2, vpcID *string, scpConfigFileName string) {
	res, err := ec2client.DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("vpc-id"),
				Values: []*string{
					vpcID,
				},
			},
		},
	})
	ac.checkError(err)

	scpConfigBytes, err := ioutil.ReadFile(path.Join(ac.configPath, "scp-config-template"))
	ac.checkError(err)
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

	err = ioutil.WriteFile(path.Join(ac.configPath, scpConfigFileName), []byte(scpConfig), 0666)
	ac.checkError(err)
}

// CreateAWSKubernetesCluster - creating AWS kubernetes cluster instances
func (ac *AWSCluster) CreateAWSKubernetesCluster() {
	sess := session.Must(session.NewSession())
	iamClient := iam.New(sess)
	eksClient := eks.New(sess)
	cfClient := cloudformation.New(sess)
	ec2Client := ec2.New(sess)

	// Creating Amazon EKS Role
	eksRoleName := awsRolePrefix + ac.serviceSuffix
	eksRoleArn := ac.createEksRole(iamClient, &eksRoleName)

	// Creating Amazon EKS Cluster VPC
	clusterStackName := awsClusterStackPrefix + ac.serviceSuffix
	clusterStackOutputs := ac.createEksClusterVpc(cfClient, &clusterStackName)

	// Creating Amazon EKS Cluster
	clusterName := awsClusterPrefix + ac.serviceSuffix
	cluster := ac.createEksCluster(eksClient, &clusterName, eksRoleArn, clusterStackOutputs)

	// Creating kubeconfig file
	ac.createKubeConfigFile(cluster)

	// Creating Amazon EKS Worker Nodes
	keyPairName := awsKeyPairPrefix + ac.serviceSuffix
	ac.createEksEc2KeyPair(ec2Client, &keyPairName)

	nodesStackName := awsNodesStackPrefix + ac.serviceSuffix
	nodeGroupName := awsNodeGroupPrefix + ac.serviceSuffix
	nodeSequrityGroup, nodeInstanceRole := ac.createEksWorkerNodes(cfClient, &nodesStackName, &nodeGroupName, &clusterName, &keyPairName, clusterStackOutputs)

	ac.authorizeSecurityGroupIngress(ec2Client, nodeSequrityGroup)

	// Enable worker nodes to join the cluster
	sf, err := ioutil.ReadFile(path.Join(ac.configPath, "aws-auth-cm-temp.yaml"))
	ac.checkError(err)

	f, err := ioutil.TempFile(os.TempDir(), "aws-auth-cm-temp-*.yaml")
	ac.checkError(err)

	s := string(sf)
	_, err = f.Write([]byte(strings.Replace(s, "<NodeInstanceRole>", *nodeInstanceRole, -1)))
	ac.checkError(err)

	_ = f.Close()

	ac.execCommand("kubectl", "apply", "-f", f.Name())
	_ = os.Remove(f.Name())

	ac.createSSHConfig(ec2Client, clusterStackOutputs.VpcId, "scp-config"+ac.serviceSuffix)

	ac.execCommand("kubectl", "apply", "-f", "aws-k8s-cni.yaml")
}

func (ac *AWSCluster) execCommand(name string, arg ...string) {
	log.Printf("> %s %s", name, strings.Join(arg, " "))
	cmd := exec.Command(name, arg...) // #nosec
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	ac.checkError(err)
	log.Printf(out.String())
}
