ifeq ($(GKE_CLUSTER_NAME),)
	GKE_CLUSTER_NAME := dev
endif

.PHONY: gke-start
gke-start: gcloud-check
	@if ! (gcloud container clusters list | grep -q ^${GKE_CLUSTER_NAME}); then \
		time gcloud container clusters create ${GKE_CLUSTER_NAME} --machine-type=n1-standard-2 --num-nodes=2 -q; \
		gcloud container clusters get-credentials ${GKE_CLUSTER_NAME}; \
		kubectl create clusterrolebinding cluster-admin-binding \
			--clusterrole cluster-admin \
  			--user $$(gcloud config get-value account); \
	fi

.PHONY: gke-destroy
gke-destroy: gcloud-check
	@if (gcloud container clusters list | grep -q ^${GKE_CLUSTER_NAME}); then \
		time gcloud container clusters delete ${GKE_CLUSTER_NAME} -q ; \
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