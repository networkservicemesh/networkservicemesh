Network Service Mesh release process
====================================

Naming
------

NSM releases will use semver to track API changes and to comply with Go module versioning standards.
The code names are chosen alphabetically from latin names in the list of [Star constellations](https://starchild.gsfc.nasa.gov/docs/StarChild/questions/88constellations.html)


Progress tracking
-----------------

Included features and progress tracking will be using a dedicated GitHub project. The release project is reviewed, ammended or issues deleted during the weekly WG calls.

Release procedure
The primary NSM git repo is https://github.com/networkservicemesh/networkservicemesh. All the git operations described here are applied on it. Consequently, if the main repo is split into smaller repos, git operations will be applied to all of them simultaneously.

The project will set a designated release date **D**. One calendar week before that date **D-7** a git release branch will be created. The branch name is in the format: `release-<MAJOR>.<MINOR>`.

On the designated date, the release with patchlevel `0` is released and tagged as `v<MAJOR>.<MINOR>.<PATCH>`. The next 2 calendar weeks will accommodate bug, performance and issue fixes. Then a new release patchlevel `1` is tagged and declared as stable on **D+14**.

Stable release transitions
--------------------------

The highest numbered release branch holds the stable release after it gets at least patch level ‘1’. The table below shows an example of stable branch transitioning.

|  Timeline | T0  | T0+1  | T0+2  | T0+3  | T0+4  | T0+5  |
|---|---|---|---|---|---|---|
|   | `v1.2.3 stable`  |  `v1.2.3 stable` | `v1.2.4 stable`  |   |  `v1.2.5 maint` |   |
|   |   | `v1.3.0 pre`  |   | `v1.3.1 stable`  |   | `v1.3.2 stable`  |

Release materials
-----------------

The subject of the release are the following items:

* Containers
    * Core containers
        * nsmd
        * nsmdp
    * Registry connectors
        * nsmd-k8s
    * Forwarder implementations
        * vppagent-forwarder
* Deployability artifacts
    * Yaml files
    * Helm charts
* APIs
    * gRPC specs / protobufs
    * Kubernetes CRD definitions
* Docs
    * Deployment
        * Popular public clouds
        * Private on-prem deployment
        * Writing Network Service yaml
    * NS Client and Endpoint development through the SDK
    * (Optional) Forwarder implementator’s guide

Image publishing
----------------

The NSM container images are published at [Dockerhub’s networkservicemesh workspace](https://hub.docker.com/u/networkservicemesh). The images published from the git branch master are tagged `latest`.  The images from the release branch get the same git tag i.e. `v<MAJOR>.<MINOR>.<PATCH>`. The last released image from that branch is tagged `stable`.

References
----------

* Issue(s) reference - #720
