.PHONY: gcb-build
gcb-build: $(addsuffix -build,$(addprefix gcb-,$(BUILD_CONTAINERS)))

.PHONY: gcb-%-build
gcb-%-build:
	@if [ "x${COMMIT}" == "x" ] ; then \
		COMMIT=latest; \
	fi ;\
	gcloud builds submit --config=deployments/gcb/cloudbuild.yaml --substitutions=_NAME=$*,_REPO=gcr.io/$(shell gcloud config get-value project),_TAG=$${COMMIT}; \

.PHONY: gcb-vppagent-dataplane-dev-build
gcb-vppagent-dataplane-dev-build:
	@echo Do not build vppagent-dataplane-dev

.PHONY: gcb-save
gcb-save: $(addsuffix -save,$(addprefix gcb-,$(BUILD_CONTAINERS))) ;

.PHONY: gcb-%-save
gcb-%-save: gcb-%-build ;


