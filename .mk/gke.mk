ifeq ($(GKE_CLUSTER_NAME),)
	GKE_CLUSTER_NAME := dev
endif

ifeq ($(GKE_CLUSTER_ZONE),)
	GKE_CLUSTER_ZONE := "northamerica-northeast1-a"
endif

ifeq ($(GKE_CLUSTER_TYPE),)
	GKE_CLUSTER_TYPE := "n1-standard-2"
endif

ifeq ($(GKE_CLUSTER_NUM_NODES),)
	GKE_CLUSTER_NUM_NODES := "2"
endif

ifeq ($(GKE_PROJECT_ID),)
	GKE_PROJECT_ID := "ci-management"
endif

.PHONY: gke-start
gke-start: gcloud-check
	@if ! (gcloud container clusters list --project=${GKE_PROJECT_ID} | grep -q ^${GKE_CLUSTER_NAME}); then \
		time gcloud container clusters create ${GKE_CLUSTER_NAME} --project=${GKE_PROJECT_ID} --machine-type=${GKE_CLUSTER_TYPE} --num-nodes=${GKE_CLUSTER_NUM_NODES} --zone=${GKE_CLUSTER_ZONE} -q; \
		echo "Writing config to ${KUBECONFIG}"; \
		gcloud container clusters get-credentials ${GKE_CLUSTER_NAME} --project=${GKE_PROJECT_ID} --zone=${GKE_CLUSTER_ZONE} ; \
		gcloud compute firewall-rules create allow-proxy-nsm --action ALLOW --rules tcp:80 --project=${GKE_PROJECT_ID}; \
		gcloud compute firewall-rules create allow-nsm --action ALLOW --rules tcp:5000-5100 --project=${GKE_PROJECT_ID}; \
		gcloud compute firewall-rules create allow-vxlan --action ALLOW --rules udp:4789 --project=${GKE_PROJECT_ID}; \
		kubectl create clusterrolebinding cluster-admin-binding \
			--clusterrole cluster-admin \
  			--user $$(gcloud config get-value account); \
	fi

.PHONY: gke-destroy
gke-destroy: gcloud-check
	@if (gcloud container clusters list --project=${GKE_PROJECT_ID} | grep -q ^${GKE_CLUSTER_NAME}); then \
		time gcloud container clusters delete ${GKE_CLUSTER_NAME} --project=${GKE_PROJECT_ID} --zone=${GKE_CLUSTER_ZONE} -q ; \
	fi

.PHONY: gcloud-check
gcloud-check:
	@if ! (gcloud version > /dev/null 2>&1); then \
		echo "You do not appear to have gcloud installed.  Please see: https://cloud.google.com/sdk/install for installation instructions"; \
	else \
		echo "gcloud installed"; \
	fi

.PHONY: gke-restart
gke-restart: gke-destroy gke-start;

.PHONY: gke-%-load-images
gke-%-load-images: ;


.PHONY: gke-docker-login
gke-docker-login:
	gcloud auth configure-docker --quiet


.PHONY: gke-%-push
gke-%-push: gke-docker-login
	@if [ "x$${COMMIT}" == "x" ] ; then \
		COMMIT=latest; \
	fi ;\
	if [ "x$${TAG}" == "x" ] ; then \
		TAG=latest; \
	fi ;\
    ORG=gcr.io/$(shell gcloud config get-value project); \
	docker tag $${ORG}/$*:$${COMMIT} $${ORG}/$*:$${TAG}; \
	docker push $${ORG}/$*:$${TAG}

gke-push: $(addsuffix -push,$(addprefix gke-,$(BUILD_CONTAINERS)))
	@echo $(addsuffix -push,$(addprefix gke-,$(BUILD_CONTAINERS)))


.ONESHELL:
.PHONY: gke-download-postmortem
gke-download-postmortem:
	@echo TODO: Download GKE postmortem data.

.PHONY: gke-print-kubelet-log
gke-print-kubelet-log:
	@echo TODO: Print nodes kubelet log.
