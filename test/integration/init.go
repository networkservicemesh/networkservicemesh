package nsmd_integration_tests

import "os"

const (
	containerRepoEnv = "CONTAINER_REPO"
)

var containerRepo = "networkservicemesh"
var containerRepoDefault = "networkservicemesh"

func init() {
	found := false
	containerRepo, found = os.LookupEnv(containerRepoEnv)

	if !found {
		containerRepo = containerRepoDefault
	}
}
