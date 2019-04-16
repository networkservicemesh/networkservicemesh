include .mk/vpp_agent.mk

.PHONY: gcb-build
gcb-build: $(addsuffix -build,$(addprefix gcb-,$(BUILD_CONTAINERS)))

.PHONY: gcb-%-build
gcb-%-build:
	@if [ "x${COMMIT}" == "x" ] ; then \
		COMMIT=latest; \
	fi ;\
	echo "RUNNING build with params: _NAME=$*,_REPO=gcr.io/$(shell gcloud config get-value project),_TAG=$${COMMIT},_VPP_AGENT=$${VPP_AGENT},_VPP_AGENT_DEV=$${VPP_AGENT_DEV}"; \
	gcloud builds submit --config=deployments/gcb/cloudbuild.yaml --substitutions=_NAME=$*,_REPO=gcr.io/$(shell gcloud config get-value project),_TAG=$${COMMIT},_VPP_AGENT=$${VPP_AGENT},_VPP_AGENT_DEV=$${VPP_AGENT_DEV}; \

.PHONY: gcb-save
gcb-save: $(addsuffix -save,$(addprefix gcb-,$(BUILD_CONTAINERS))) ;

.PHONY: gcb-%-delete
gcb-%-delete:
	@if [ "x${COMMIT}" == "x" ] ; then \
		COMMIT=latest --force-delete-tags; \
	fi ;\
	REPO=$(shell gcloud config get-value project); \
	echo "RUNNING image delete: _NAME=$*,_REPO=gcr.io/$${REPO},_TAG=$${COMMIT}"; \
	gcloud container images delete gcr.io/$${REPO}/$*:$${COMMIT} --quiet  || true

.PHONY: gcb-delete-build
gcb-delete-build: $(addsuffix -build-delete,$(addprefix gcb-,$(BUILD_CONTAINERS)))

.PHONY: gcb-delete
gcb-delete: $(addsuffix -delete,$(addprefix gcb-,$(BUILD_CONTAINERS))) gcb-delete-build

.PHONY: gcb-%-save
gcb-%-save: gcb-%-build ;


