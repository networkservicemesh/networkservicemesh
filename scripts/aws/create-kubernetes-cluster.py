#!/usr/bin/env python

import boto3
import os
import tempfile
import time

from botocore.exceptions import ClientError

eks_client = boto3.client('eks')
cf_client = boto3.client('cloudformation')
iam_client = boto3.client('iam')
ec2_client = boto3.client('ec2')

def create_eks_role(eks_role_name):
    print("Creating EKS service role \"%s\"..." % eks_role_name)

    rpf = open(os.path.dirname(__file__) + "/amazon-eks-role-policy.json")
    try:
        iam_client.create_role(
            RoleName = eks_role_name,
            Path = "/",
            Description = "Allows EKS to manage clusters on your behalf.",
            AssumeRolePolicyDocument = rpf.read()

        )
        iam_client.attach_role_policy(
            RoleName = eks_role_name,
            PolicyArn = 'arn:aws:iam::aws:policy/AmazonEKSClusterPolicy'
        )
        iam_client.attach_role_policy(
            RoleName = eks_role_name,
            PolicyArn = 'arn:aws:iam::aws:policy/AmazonEKSServicePolicy'
        )
    except ClientError as e:
        if e.response['Error']['Code'] == "EntityAlreadyExists":
            print("Warning: %s" % e.response['Error']['Message'])
        else:
            print("Error: %s" % e.response['Error']['Message'])
            exit(1)
    finally:
        rpf.close()

    eks_role_arn = iam_client.get_role(RoleName = eks_role_name)["Role"]["Arn"]
    print("Role \"%s\"(%s) successfully created!" % (eks_role_name, eks_role_arn))
    return eks_role_arn

def create_eks_cluster_vpc(cluster_stack_name):

    print("Creating Amazon EKS Cluster VPC \"%s\"..." % cluster_stack_name)

    sf = open(os.path.dirname(__file__) + "/amazon-eks-vpc.yaml", "r")
    try:
        cf_client.create_stack(
            StackName=cluster_stack_name,
            TemplateBody=sf.read()
        )
    except ClientError as e:
        if e.response['Error']['Code'] != "AlreadyExistsException":
            print("Error: %s" % e.response['Error']['Message'])
            exit(1)
        print("Warning: %s" % e.response['Error']['Message'])
    finally:
        sf.close()

    while True:
        try:
            response = cf_client.describe_stacks(StackName=cluster_stack_name)
        except ClientError as e:
            print("Error: %s" % e.response['Error']['Message'])
            exit(1)

        if response[u'Stacks'][0][u'StackStatus'] == "CREATE_COMPLETE":
            break
        elif response[u'Stacks'][0][u'StackStatus'] == "CREATE_IN_PROGRESS":
            time.sleep(1)
        else:
            print("Error: Unexpected stack status: %s" % response[u'Stacks'][0][u'StackStatus'])
            exit(1)

    print("Cluster VPC \"%s\" successfully created!" % cluster_stack_name)

    #### Fill the map with required stack options
    cluster_stack_outputs = {}
    for option in response[u'Stacks'][0][u'Outputs']:
        cluster_stack_outputs[option[u'OutputKey']] = option[u'OutputValue']

    return cluster_stack_outputs

def create_eks_cluster(cluster_name, eks_role_arn, cluster_stack_outputs):
    print("Creating Amazon EKS Cluster \"%s\"..." % cluster_name)
    try:
        eks_client.create_cluster(
            name = cluster_name,
            roleArn = eks_role_arn,
            resourcesVpcConfig = {
                'subnetIds': cluster_stack_outputs['SubnetIds'].split(","),
                'securityGroupIds': [
                    cluster_stack_outputs['SecurityGroups']
                ],
                'endpointPublicAccess': True,
                'endpointPrivateAccess': False
            }
        )
    except ClientError as e:
        if e.response['Error']['Code'] != "ResourceInUseException":
            print("Error: %s" % e.response['Error']['Message'])
            exit(1)
        print("Warning: %s" % e.response['Error']['Message'])

    while True:
        try:
            response = eks_client.describe_cluster(name=cluster_name)
        except ClientError as e:
            print("Error: %s" % e.response['Error']['Message'])
            exit(1)

        if response[u'cluster'][u'status'] == "ACTIVE":
            break
        elif response[u'cluster'][u'status'] == "CREATING":
            time.sleep(1)
        else:
            print("Error: Unexpected cluster status: %s" % response[u'cluster'][u'status'])
            exit(1)

    print("EKS Cluster \"%s\" successfully created!" % cluster_name)

def create_eks_ec2_key_pair(key_name):
    print("Creating Amazon EC2 key pair \"%s\"..." % key_name)

    try:
        ec2_client.create_key_pair(
            KeyName=key_name
        )
    except ClientError as e:
        if e.response['Error']['Code'] != "InvalidKeyPair.Duplicate":
            print("Error: %s" % e.response['Error']['Message'])
            exit(1)
        print("Warning: %s" % e.response['Error']['Message'])

    print("Amazon EC2 key pair \"%s\" successfully created!" % key_name)

def create_eks_worker_nodes(nodes_stack_name, node_group_name, cluster_name, key_pair_name, cluster_stack_outputs):
    print("Creating Amazon EKS Worker Nodes on cluster \"%s\"..." % cluster_name)

    sf = open(os.path.dirname(__file__) + "/amazon-eks-nodegroup.yaml", "r")
    try:
        cf_client.create_stack(
            StackName=nodes_stack_name,
            TemplateBody=sf.read(),
            Capabilities=["CAPABILITY_IAM"],
            Parameters=[
                {
                    'ParameterKey': 'KeyName',
                    'ParameterValue': key_pair_name,
                },
                {
                    'ParameterKey': 'NodeImageId',
                    'ParameterValue': "ami-0484545fe7d3da96f",
                },
                {
                    'ParameterKey': 'ClusterName',
                    'ParameterValue': cluster_name,
                },
                {
                    'ParameterKey': 'NodeGroupName',
                    'ParameterValue': node_group_name,
                },
                {
                    'ParameterKey': 'ClusterControlPlaneSecurityGroup',
                    'ParameterValue': cluster_stack_outputs["SecurityGroups"],
                },
                {
                    'ParameterKey': 'VpcId',
                    'ParameterValue': cluster_stack_outputs["VpcId"],
                },
                {
                    'ParameterKey': 'Subnets',
                    'ParameterValue': cluster_stack_outputs["SubnetIds"],
                }
            ]
        )
    except ClientError as e:
        if e.response['Error']['Code'] != "AlreadyExistsException":
            print("Error: %s" % e.response['Error']['Message'])
            exit(1)
        print("Warning: %s" % e.response['Error']['Message'])
    finally:
        sf.close()

    while True:
        try:
            response = cf_client.describe_stacks(StackName=nodes_stack_name)
        except ClientError as e:
            print("Error: %s" % e.response['Error']['Message'])
            exit(1)

        if response[u'Stacks'][0][u'StackStatus'] == "CREATE_COMPLETE":
            break
        elif response[u'Stacks'][0][u'StackStatus'] == "CREATE_IN_PROGRESS":
            time.sleep(1)
        else:
            print("Error: Unexpected stack status: %s" % response[u'Stacks'][0][u'StackStatus'])
            exit(1)


    print("EKS Worker Nodes \"%s\" successfully created!" % nodes_stack_name)

    #### Fill the map with required stack options
    nodes_stack_outputs = {}
    for option in response[u'Stacks'][0][u'Outputs']:
        nodes_stack_outputs[option[u'OutputKey']] = option[u'OutputValue']

    return nodes_stack_outputs

def main():
    suffix = os.environ.get("NSM_AWS_SERVICE_SUFFIX", "")

    #### Creating Amazon EKS Role
    eks_role_name = "nsm-role" + suffix
    eks_role_arn = create_eks_role(eks_role_name)

    #### Creating Amazon EKS Cluster VPC
    cluster_stack_name = 'nsm-srv' + suffix
    cluster_stack_outputs = create_eks_cluster_vpc(cluster_stack_name)

    #### Creating Amazon EKS Cluster
    cluster_name = "nsm" + suffix
    create_eks_cluster(cluster_name, eks_role_arn, cluster_stack_outputs)

    cmd = "aws eks update-kubeconfig --name %s" % cluster_name
    print("> " + cmd)
    os.system(cmd)

    #### Creating Amazon EKS Worker Nodes
    key_pair_name = "nsm-key-pair" + suffix
    create_eks_ec2_key_pair(key_pair_name)

    nodes_stack_name = "nsm-nodes" + suffix
    node_group_name = "nsm-node-group" + suffix
    nodes_stack_outputs = create_eks_worker_nodes(nodes_stack_name, node_group_name, cluster_name, key_pair_name, cluster_stack_outputs)

    #### Enable worker nodes to join the cluster
    aws_auth_fname_temp = os.path.dirname(__file__) + "/aws-auth-cm-temp.yaml"
    aws_auth_fname = tempfile.gettempdir() + "/aws-auth-cm-temp.yaml"
    os.system("sed -e 's;rolearn: .*;rolearn: %s;g' %s > %s" % (nodes_stack_outputs["NodeInstanceRole"], aws_auth_fname_temp, aws_auth_fname))
    os.system("kubectl apply -f %s" % aws_auth_fname)
    os.remove(aws_auth_fname)

if __name__ == '__main__':
    main()
