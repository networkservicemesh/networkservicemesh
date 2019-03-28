# Register inside Google Cloud Platform

# Prerequisites

## Install Google Cloud SDK.

* [Quick start guide](https://cloud.google.com/sdk/docs/quickstarts)

### OSX via brew cask

```shell
brew cask install google-cloud-sdk
```

## Use Google Cloud environment.

Before start using of GKE, please execute the following command:

```shell
source ./.env/gke.env
```

# Configure Google cloud SDK tool 

## 0. Authentication

[Authentication](https://cloud.google.com/sdk/gcloud/reference/auth/login)

```shell
google auth login
```
 
## 1. Select a project.

* List projects

```shell
gcloud projects list
```

* Set the active project

```shell
gcloud config set project myProject
```

## 2. Specify area

* List zones/regions

```shell
gcloud compute regions list
gcloud compute zones list
```

* Select default zone

```shell
gcloud config set compute/zone europe-west1-c
```

* Select default region

```shell
gcloud config set compute/zone europe-west1
```

(!) Be sure to specify zone, not region, since it will create more nodes.

So at this moment we are ready to use google cloud platform. 

# Build

# Build all nodes using google cloud build.

* Build everything

```shell
make k8s-save -j20
```
