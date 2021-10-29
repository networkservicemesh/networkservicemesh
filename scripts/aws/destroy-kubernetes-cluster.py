#!/usr/bin/env python

import boto3
import os
import sys
import time

from botocore.exceptions import ClientError
from threading import Thread

eks_client = boto3.client('eks')
cf_client = boto3.client('cloudformation')
iam_client = boto3.client('iam')
ec2_client = boto3.client('ec2')

def delete_eks_role(eks_role_name):
    print("Deleting EKS service role \"%s\"..." % eks_role_name)

    try:
        iam_client.detach_role_policy(
            RoleName = eks_role_name,
            PolicyArn = 'arn:aws:iam::aws:policy/AmazonEKSClusterPolicy'
        )
    except ClientError as e:
        print("Warning: %s" % e.response['Error']['Message'])

    try:
        iam_client.detach_role_policy(
            RoleName = eks_role_name,
            PolicyArn = 'arn:aws:iam::aws:policy/AmazonEKSServicePolicy'
        )
    except ClientError as e:
        print("Warning: %s" % e.response['Error']['Message'])

    try:
        iam_client.delete_role(
            RoleName = eks_role_name
        )
    except ClientError as e:
        print("Error: %s" % e.response['Error']['Message'])
        return

    print("Role \"%s\" successfully deleted!" % (eks_role_name))

def delete_eks_cluster_vpc(cluster_stack_name):

    sys.stdout.write("Deleting Amazon EKS Cluster VPC \"%s\"...\n" % cluster_stack_name)

    try:
        cf_client.delete_stack(
            StackName=cluster_stack_name
        )
    except ClientError as e:
        print("Error: %s" % e.response['Error']['Message'])
        return

    while True:
        try:
            response = cf_client.describe_stacks(StackName=cluster_stack_name)
        except ClientError as e:
            print("Error: %s" % e.response['Error']['Message'])
            return

        if response[u'Stacks'][0][u'StackStatus'] == "DELETE_COMPLETE":
            break
        elif response[u'Stacks'][0][u'StackStatus'] == "DELETE_IN_PROGRESS":
            time.sleep(1)
        else:
            print("Error: Unexpected stack status: %s" % response[u'Stacks'][0][u'StackStatus'])
            return

    print("Cluster VPC \"%s\" successfully deleted!" % cluster_stack_name)

def delete_eks_cluster(cluster_name):
    sys.stdout.write("Deleting Amazon EKS Cluster \"%s\"...\n" % cluster_name)
    try:
        eks_client.delete_cluster(
            name = cluster_name,
        )
    except ClientError as e:
        print("Error: %s" % e.response['Error']['Message'])
        return

    while True:
        try:
            response = eks_client.describe_cluster(name=cluster_name)
        except ClientError as e:
            break

        if response[u'cluster'][u'status'] == "DELETING":
            time.sleep(1)
        else:
            print("Error: Unexpected cluster status: %s" % response[u'cluster'][u'status'])
            return

    print("EKS Cluster \"%s\" successfully deleted!" % cluster_name)

def delete_eks_ec2_key_pair(key_name):
    print("Deleting Amazon EC2 key pair \"%s\"..." % key_name)

    try:
        ec2_client.delete_key_pair(
            KeyName=key_name
        )
    except ClientError as e:
        print("Error: %s" % e.response['Error']['Message'])
        return

    print("Amazon EC2 key pair \"%s\" successfully deleted!" % key_name)

def delete_eks_worker_nodes(nodes_stack_name):
    sys.stdout.write("Deleting Amazon EKS Worker Nodes...\n")

    try:
        cf_client.delete_stack(
            StackName=nodes_stack_name,
        )
    except ClientError as e:
        print("Error: %s" % e.response['Error']['Message'])
        return

    while True:
        try:
            response = cf_client.describe_stacks(StackName=nodes_stack_name)
        except ClientError as e:
            print("Error: %s" % e.response['Error']['Message'])
            return

        if response[u'Stacks'][0][u'StackStatus'] == "DELETE_COMPLETE":
            break
        elif response[u'Stacks'][0][u'StackStatus'] == "DELETE_IN_PROGRESS":
            time.sleep(1)
        else:
            print("Error: Unexpected stack status: %s" % response[u'Stacks'][0][u'StackStatus'])
            return

    print("EKS Worker Nodes \"%s\" successfully deleted!" % nodes_stack_name)

def main():
    suffix = os.environ.get("NSM_AWS_SERVICE_SUFFIX", "")

    #### Deleting Amazon EKS Worker Nodes
    nodes_stack_name = "nsm-nodes" + suffix
    delete_eks_worker_nodes_thread = Thread(
        target = delete_eks_worker_nodes,
        args = (nodes_stack_name,)
    )
    delete_eks_worker_nodes_thread.start()

    #### Deleting Amazon EKS Cluster
    cluster_name = "nsm" + suffix
    delete_eks_cluster_thread = Thread(
        target = delete_eks_cluster,
        args = (cluster_name,)
    )
    delete_eks_cluster_thread.start()

    #### Deleting Amazon EKS Cluster VPC
    cluster_stack_name = 'nsm-srv' + suffix
    delete_eks_cluster_vpc_thread = Thread(
        target = delete_eks_cluster_vpc,
        args = (cluster_stack_name,)
    )
    delete_eks_cluster_vpc_thread.start()

    #### Wait for deletion complete
    delete_eks_worker_nodes_thread.join()
    delete_eks_cluster_thread.join()
    delete_eks_cluster_vpc_thread.join()

    #### Deleting keys and roles
    key_pair_name = "nsm-key-pair" + suffix
    delete_eks_ec2_key_pair(key_pair_name)
    eks_role_name = "nsm-role" + suffix
    delete_eks_role(eks_role_name)

if __name__ == '__main__':
    main()
