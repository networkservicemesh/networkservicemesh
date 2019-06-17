package config

type DeviceConfig struct {
	Plan            string `yaml:"plan"` // Plan
	OperatingSystem string `yaml:"os"`   // Operating system
	BillingCycle    string `yaml:"billing_cycle"`
	Name            string `yaml:"name"` // Host name prefix, will create ENV variable IP_HostName
}

type PacketConfig struct {
	Devices           []*DeviceConfig `yaml:"devices"`            // A set of device configuration required to be created before starting cluster.
	Facilities        []string        `yaml:"facilities"`         // A set of facility filters
	PreferredFacility string          `yaml:"preferred-facility"` // A prefered facility key
	SshKey            string          `yaml:"ssh-key"`            // A location of ssh key
}

type ClusterProviderConfig struct {
	Name       string            `yaml:"name"`       // name of provider, GKE, Azure, etc.
	Kind       string            `yaml:"kind"`       // register provider type, 'generic', 'gke', etc.
	Instances  int               `yaml:"instances"`  // Number of required instances, executions will be split between instances.
	Timeout    int               `yaml:"timeout"`    // Timeout for start, stop
	RetryCount int               `yaml:"retry"`      // A count of start retrying steps.
	NodeCount  int               `yaml:"node-count"` // A count of nodes should be available via API to match cluster is alive.
	StopDelay  int64             `yaml:"stop-delay"` // A timeout after stop and starting of session again.
	Enabled    bool              `yaml:"enabled"`    // Is it enabled by default or not
	Parameters map[string]string `yaml:"parameters"` // A parameters specific for provider
	Scripts    map[string]string `yaml:"scripts"`    // A parameters specific for provider
	Env        []string          `yaml:"env"`        // Extra environment variables
	EnvCheck   []string          `yaml:"env-check"`  // Check if environment has required environment variables present.

	Packet *PacketConfig `yaml:"packet"` // A special packet configuration section
}

type ExecutionConfig struct {
	// Executions, every execution execute some tests agains configured set of clusters
	Name            string   `yaml:"name"`           // Execution name
	Tags            []string `yaml:"tags"`           // A list of tags for this configured execution.
	PackageRoot     string   `yaml:"root"`           // A package root for this test execution, default .
	Timeout         int64    `yaml:"timeout"`        // Invidiaul test timeout, "60" passed to gotest, in seconds
	ExtraOptions    []string `yaml:"extra-options"`  // Extra options to pass to gotest
	ClusterCount    int      `yaml:"cluster-count"`  // A number of clusters required for this execution, default 1
	KubernetesEnv   []string `yaml:"kubernetes-env"` // Names of environment variables to put cluster names inside.
	ClusterSelector []string `yaml:"selector"`       // A cluster name to execute this tests on.
	// Multi Cluster tests
}

type CloudTestConfig struct {
	Version    string                   `yaml:"version"` // Provider file version, 1.0
	Providers  []*ClusterProviderConfig `yaml:"providers"`
	ConfigRoot string                   `yaml:"root"` // A provider stored configurations root.
	Reporting  struct {
		// A junit report location.
		JUnitReportFile string `yaml:"junit-report"`
	} `yaml:"reporting"`

	Executions []*ExecutionConfig `yaml:"executions"`
	Timeout    int64              `yaml:"timeout"` // Global timeout in minutes
}
