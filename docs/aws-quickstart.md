# Register inside Google Cloud Platform

## Create New Access Key

[Create New Access Key](https://console.aws.amazon.com/iam/home?region=us-east-2#/security_credentials)

Put key ID and secret into environment variables:
* NSM_AWS_ACCESS_KEY_ID
* NSM_AWS_SECRET_ACCESS_KEY

# Prerequisites

## Setup AWS.

```shell
make aws-init
```

## Use AWS environment.

Before start using of AWS, please execute the following command:

```shell
source ./.env/aws.env
```

## Specify AWS services instances names

Set NSM_AWS_SERVICE_SUFFIX environment variable to specify AWS services instances names

# Usage

## Create cluster and configure kubernetes

```shell
make aws-start
```

## Configure all nodes.

```shell
make k8s-config
```

## Finish working

Release all AWS services

```shell
make aws-destroy
```
