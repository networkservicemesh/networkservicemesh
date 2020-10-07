ifeq ($(AZURE_RESOURCE_GROUP),)
AZURE_RESOURCE_GROUP := nsm-ci
endif

ifeq ($(AZURE_CLUSTER_NAME),)
AZURE_CLUSTER_NAME := nsm-ci-cluster
endif

ifeq ($(AZURE_CREDENTIALS_PATH),)
AZURE_CREDENTIALS_PATH := scripts/azure/.kube/config
endif

.PHONY: azure-start
azure-start: azure-check
	@scripts/azure/create-aks-cluster.sh "$(AZURE_RESOURCE_GROUP)" "$(AZURE_CLUSTER_NAME)" "$(AZURE_CREDENTIALS_PATH)" "$(AZURE_SERVICE_PRINCIPAL)" "$(AZURE_SERVICE_PRINCIPAL_SECRET)"

.PHONY: azure-destroy
azure-destroy: azure-check
	@scripts/azure/destroy-aks-cluster.sh "$(AZURE_RESOURCE_GROUP)" "$(AZURE_CLUSTER_NAME)"

.PHONY: azure-check
azure-check: azure-cli-check azure-group-check

.PHONY: azure-cli-check
azure-cli-check:
	@echo -n "Checking for Azure CLI tool..."
	@if (which az > /dev/null 2>&1); then \
		echo "installed"; \
	else \
		echo "not found"; \
		echo "You don't appear to have Azure CLI tool installed.  Please see: https://docs.microsoft.com/en-us/cli/azure/install-azure-cli?view=azure-cli-latest for installation instructions"; \
		false; \
	fi

.PHONY: azure-group-check
azure-group-check: azure-cli-check
	@echo -n "Checking for resource group $(AZURE_RESOURCE_GROUP)..."
	@if [ "$$(az group exists --name $(AZURE_RESOURCE_GROUP))" = "true" ]; then \
		echo "exists"; \
	else \
		echo "not found"; \
		false; \
	fi

.PHONY: azure-download-postmortem
azure-download-postmortem:
	@echo "Not implemented yet."

.PHONY: azure-print-kubelet-log
azure-print-kubelet-log:
	@echo TODO: Print nodes kubelet log.
