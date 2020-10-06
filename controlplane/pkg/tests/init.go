package tests

import (
	"os"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

func init() {
	_ = os.Setenv(tools.InsecureEnv, "true")
}
