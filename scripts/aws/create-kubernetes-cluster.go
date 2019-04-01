package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"runtime"
)

import (
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/iam"
)

var _, currentFilePath, _, _ = runtime.Caller(0)
var currentPath = path.Dir(currentFilePath)

func check_error(err error) {
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if (aerr.Code() != "EntityAlreadyExists") {
				fmt.Printf("Error (%s): %s\n", aerr.Code(), aerr.Message())
				os.Exit(1)
			}
		} else {
			fmt.Printf("Error: %s\n", err.Error())
			os.Exit(1)
		}
	}
}

func create_eks_role(iam_client *iam.IAM, eks_role_name *string) (*string) {
	fmt.Printf("Creating EKS service role \"%s\"...\n", eks_role_name)
	roleDescription := "Allows EKS to manage clusters on your behalf."

	rpf, err := ioutil.ReadFile(path.Join(currentPath, "amazon-eks-role-policy.json"))
	check_error(err)

	rps := string(rpf)
	_, err = iam_client.CreateRole(&iam.CreateRoleInput{
		RoleName: eks_role_name,
		Description: &roleDescription,
		AssumeRolePolicyDocument: &rps,
			})
	check_error(err)

	policyArn := "arn:aws:iam::aws:policy/AmazonEKSClusterPolicy"
	_, err = iam_client.AttachRolePolicy(&iam.AttachRolePolicyInput{
		RoleName: eks_role_name,
		PolicyArn: &policyArn,
			})
	check_error(err)

	policyArn = "arn:aws:iam::aws:policy/AmazonEKSServicePolicy"
	_, err = iam_client.AttachRolePolicy(&iam.AttachRolePolicyInput{
		RoleName: eks_role_name,
		PolicyArn: &policyArn,
	})
	check_error(err)


	result, err := iam_client.GetRole(&iam.GetRoleInput{
		RoleName: eks_role_name,
		})
	check_error(err)

	return result.Role.Arn
}

func create_eks_cluster_vpc(iam_client *iam.IAM, eks_role_name *string) {

}

func main() {
	sess := session.Must(session.NewSession())
	iam_client := iam.New(sess)
	eks_client := eks.New(sess)
	cf_client := cf.New(sess)
	ec2_client := ec2.New(sess)

	service_suffix := os.Getenv("NSM_AWS_SERVICE_SUFFIX")
	eks_role_name := "nsm-role" + service_suffix
	eks_role_arn := *create_eks_role(iam_client, &eks_role_name)



	fmt.Printf("%s\n", eks_role_arn)
}