# Extending Network Service Mesh (NSM)


# Introduction

This document is initially aimed at developers looking to extend the NSM with new plugins to provide users with new functionality. It is based on the version of NSM available August 2018 in the [Network Service Mesh Github](https://github.com/ligato/networkservicemesh) Repository. It documents the steps that were used to create the simple-dataplane extension. 

All steps assume the top level directory is networkservicemesh from github.


# Development Steps

The development steps are to add a new data plane called cnf-dataplane.


## Edit Makefile

The top level Makefile [networkservicemesh/Makefile] to add to the following targets

docker-build: add "docker-build-cnf-dataplane"

docker-push: add "docker-push-cnf-dataplane"


## Dockerfile

A new dockerfile is added: networkservicemesh/build/Dockerfile-cnf.dataplane


## API Directory and Interface Definition

Create a new directory for the cnf-dataplane daemonset CRD interface definition.

The new directory is: networkservicemesh/pkg/nsm/apis/cnfdataplane

The interface definition file written in Google Protobufs is added to this directory. 

The file will be:  networkservicemesh/pkg/nsm/apis/cnfdataplane/cnfdataplane.proto

The interface definitions must match the go code created for the daemonset.


## Documentation

A documentation file should be added to the documentation directory: networkservicemesh/docs.

The suggested naming for the documentation file is:

    networkservicemesh/docs/README-cnfdataplane.md

## DaemonSet Code

This is the core file to implement the networking required for the plugin. It is deployed as a DaemonSet and the NSM infrastructure will call the entry points when a YAML file is deployed that activates the NSM infrastructure.

A new directory should be created: networkservicemesh/cmd/cnf-dataplane

A new file is created in the directory: networkservicemesh/cmd/cnf-dataplane/cnd-dataplane.go

## YAML Files

### DaemonSet YAML File
To activate the NSM plugin end users need to create a YAML file to deploy the DaemonSet. 

This file should be created: networkservicemesh/conf/sample/vnf-dataplane.yaml

This will define the DaemonSet to be run in each Pod.

### End User YAML File
To use the plugin the end user will need to add the following to their Pod Yaml file. Need to provide an example how to integrate into a simple application.

**Todo not sure about this step**