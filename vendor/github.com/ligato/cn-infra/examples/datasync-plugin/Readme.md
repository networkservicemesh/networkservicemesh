# Datasync example

### Requirements

To start the example you have to have ETCD running first.
if you don't have it installed locally you can use the following docker
image.
```
sudo docker run -p 2379:2379 --name etcd --rm \
    quay.io/coreos/etcd:v3.0.16 /usr/local/bin/etcd \
    -advertise-client-urls http://0.0.0.0:2379 \
    -listen-client-urls http://0.0.0.0:2379
```

It will bring up ETCD listening on port 2379 for client communication.

### Usage

In the example, the location of the ETCD configuration file is defined
with the `-etcd-config` argument or through the `ETCD_CONFIG`
environment variable.
By default, the application will try to search for `etcd.conf`
in the current working directory.
If the configuration file cannot be loaded, the initialization
of the etcd plugin will be skipped and the example scenario will thus
not execute in its entirety.

To run the example, type:
```
go run main.go deps.go [-etcd-config <config-filepath>]
```

