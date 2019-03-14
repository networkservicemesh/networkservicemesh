# Register inside Google Cloud Platform

# Pre requisites
## Install Google Cloud SDK.

* [Quick start guides](https://cloud.google.com/sdk/docs/quickstarts)

### Macos via brew cask
`brew cask install google-cloud-sdk`

## Use Google Cloud environment.

Before start using of GKE please execute following command:

`source ./.env/gke.env`

# Configure Google cloud SDK tool 

## 0. Authentication

`google auth login` - [reference](https://cloud.google.com/sdk/gcloud/reference/auth/login)
 
## 1. Select a project.

* list projects - `gcloud projects list`
* set active project - `gcloud config set project myProject`

## 2. Specify area
* list zones/regions - `gcloud compute regions list`
* select default zone - `gcloud config set compute/zone asia-east2-a`

So at this moment we are ready to use google cloud platform. 

# Build

# Build all nodes using google cloud build.
* `make k8s-save -j20`

