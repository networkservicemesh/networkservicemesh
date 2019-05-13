#!/usr/bin/env bash

# Install latest stable kubectl
if ! command -v kubectl > /dev/null 2>&1; then
    curl -LO https://storage.googleapis.com/kubernetes-release/release/"$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)"/bin/linux/amd64/kubectl
    chmod +x "kubectl" && sudo mv "kubectl" /usr/local/bin/
fi
