package providers

import (
	"time"

	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/config"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/execmanager"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/k8s"
)

// InstanceOptions - instance operation parameters
type InstanceOptions struct {
	NoInstall        bool
	NoPrepare        bool
	NoMaskParameters bool
	NoStop           bool
}

// ClusterInstance - Instanceof of one cluster
// Some of cluster cloud be alive by default, it could bare metal cluster,
// and we do not need to perform any startup, shutdown code on them.
type ClusterInstance interface {
	// Return cluster Kubernetes configuration file .config location.
	GetClusterConfig() (string, error)

	// Perform startup of cluster
	Start(timeout time.Duration) (string, error)
	// Destroy cluster
	// Should destroy cluster with timeout passed, if time is left should report about error.
	Destroy(timeout time.Duration) error

	// Return root folder to store test artifacts associated with this cluster
	GetRoot() string

	// Is cluster is running right now
	IsRunning() bool
	CheckIsAlive() error
	GetID() string
}

// ClusterProvider - provides operations with clusters
type ClusterProvider interface {
	// CreateCluster - Create a cluster based on parameters
	// CreateCluster - Creates a cluster instance and put Kubernetes config file into clusterConfigRoot
	// could fully use clusterConfigRoot folder for any temporary files related to cluster.
	CreateCluster(config *config.ClusterProviderConfig, factory k8s.ValidationFactory,
		manager execmanager.ExecutionManager,
		instanceOptions InstanceOptions) (ClusterInstance, error)

	// ValidateConfig - Check if config are valid and all parameters required by this cluster are fit.
	ValidateConfig(config *config.ClusterProviderConfig) error
}

// ClusterProviderFunction - function type to create cluster provider
type ClusterProviderFunction func(root string) ClusterProvider
