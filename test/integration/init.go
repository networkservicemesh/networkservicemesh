package nsmd_integration_tests

import "os"

const (
	containerRepoEnv     = "CONTAINER_REPO"
	containerRepoDefault = "networkservicemesh"
)

func GetContainerRepo() string {
	containerRepo, found := os.LookupEnv(containerRepoEnv)

	if !found {
		containerRepo = containerRepoDefault
	}
	return containerRepo
}
