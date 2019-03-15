# Register inside Google Cloud Platform

# Pre requisites
## Install Google Cloud SDK.

* [Quick start guides](https://cloud.google.com/sdk/docs/quickstarts)

### Macos via brew cask
`shell
brew cask install google-cloud-sdk
`

## Use Google Cloud environment.

Before start using of GKE please execute following command:

`shell
source ./.env/gke.env
`

# Configure Google cloud SDK tool 

## 0. Authentication

`shell
google auth login
` - [reference](https://cloud.google.com/sdk/gcloud/reference/auth/login)
 
## 1. Select a project.

* list projects - 
`shell
gcloud projects list
`
* set active project - 
`shell
gcloud config set project myProject
`

## 2. Specify area
* list zones/regions - 
`shell
gcloud compute regions list
`
* select default zone - 
`shell
gcloud config set compute/zone asia-east2-a
`

So at this moment we are ready to use google cloud platform. 

# Build

# Build all nodes using google cloud build.
* Build all stuff 
`shell
make k8s-save -j20
`