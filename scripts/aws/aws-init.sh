#!/bin/bash

pip install awscli boto3 --upgrade --user

aws configure set aws_access_key_id "$NSM_AWS_ACCESS_KEY_ID"
aws configure set aws_secret_access_key "$NSM_AWS_SECRET_ACCESS_KEY"
aws configure set default.region us-east-2