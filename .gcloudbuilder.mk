
.PHONY: gcloud-build
gcloud-build: $(addsuffix -build,$(addprefix gcloud-,$(BUILD_CONTAINERS)))

.PHONY: gcloud-%-build
gcloud-%-build:
	@if [ "x${COMMIT}" == "x" ] ; then \
		COMMIT=latest; \
	fi ;\
	gcloud builds submit --config=gke/cloudbuild.yaml --substitutions=_NAME=$*,_REPO=$(CONTAINER_REPO),_TAG=$${COMMIT}; \

.PHONY: gcloud-save
gcloud-save: $(addsuffix -save,$(addprefix gcloud-,$(BUILD_CONTAINERS)));

.PHONY: gcloud-%-save
gcloud-%-save: gcloud-%-build;


