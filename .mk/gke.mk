.PHONY: gke-start
gke-start: gcloud-check
	@if ! (gcloud container clusters list | grep -q ^dev); then \
		time gcloud container clusters create dev --machine-type=n1-standard-2 --num-nodes=2 -q; \
		gcloud container clusters get-credentials dev; \
		kubectl create clusterrolebinding cluster-admin-binding \
			--clusterrole cluster-admin \
  			--user $$(gcloud config get-value account); \
	fi

.PHONY: gke-destroy
gke-destroy: gcloud-check
	@if (gcloud container clusters list | grep -q ^dev); then \
		time gcloud container clusters delete dev -q ; \
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