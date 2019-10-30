# Network Service Mesh - Prerequisites for OSX

## Preparing an OSX host to run Network Service Mesh

The following instructions assume OSX environment

## Docker

Install [Docker desktop](https://www.docker.com/products/docker-desktop) from the official packages.

> Please ensure that Docker is started, or optionally select "Start Docker Desktop when you log in" on the General tab in Docker's Preferences.

## Kubectl

The rest of the prerequisites can be easily installed using `brew`:

```bash
brew install kubectl
```

## VirtualBox

```bash
brew cask install virtualbox
```

## Vagrant

```bash
brew cask install vagrant
```

## Protobuf code generation tools.

```bash
./scripts/prepare_generate.sh
```