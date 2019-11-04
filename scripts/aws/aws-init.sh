#!/bin/bash

mkdir -p "$HOME/.aws"
mkdir -p "$HOME/bin"

echo "[default]
region = us-east-2" > "$HOME/.aws/config"

echo "[default]
aws_access_key_id = $AWS_ACCESS_KEY_ID
aws_secret_access_key = $AWS_SECRET_ACCESS_KEY" > "$HOME/.aws/credentials"

if [ ! -f "$HOME/bin/aws-iam-authenticator" ]; then
    unameOut="$(uname -s)"
    case "${unameOut}" in
        Linux*)     curl -o "$HOME/bin/aws-iam-authenticator" https://amazon-eks.s3-us-west-2.amazonaws.com/1.12.7/2019-03-27/bin/linux/amd64/aws-iam-authenticator;;
        Darwin*)    curl -o "$HOME/bin/aws-iam-authenticator" https://amazon-eks.s3-us-west-2.amazonaws.com/1.12.7/2019-03-27/bin/darwin/amd64/aws-iam-authenticator;;
        *)          echo "Unknown uname $unameOut" && exit 1
    esac
fi
chmod +x "$HOME/bin/aws-iam-authenticator"
export PATH="$HOME/bin:$PATH"
