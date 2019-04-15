#!/usr/bin/env bash

KUBECTL_VERSION=v1.14.1

# Install kubectl
if ! command -v kubectl > /dev/null 2>&1; then
	curl -Lo kubectl https://storage.googleapis.com/kubernetes-release/release/"${KUBECTL_VERSION}"/bin/linux/amd64/kubectl && \
		chmod +x "kubectl" && sudo mv "kubectl" /usr/local/bin/
fi