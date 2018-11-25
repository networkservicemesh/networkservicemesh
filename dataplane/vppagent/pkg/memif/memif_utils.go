package memif

import (
	"github.com/sirupsen/logrus"
	"os"
	"path"
	"strconv"

	local "github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/connection"
)

const (
	MemifBaseDirectory    = "/memif"
	DefaultSocketFilename = "memif.sock"
)

func CreateDirectory(path string) error {
	if err := os.MkdirAll(path, 0777); err != nil {
		return err
	}
	logrus.Infof("Create directory: %s", path)
	return nil
}

func BuildSocketPath(mechanism *local.Mechanism) string {
	socketFilename, ok := mechanism.Parameters[local.SocketFilename]
	if !ok {
		socketFilename = DefaultSocketFilename
	}
	return path.Join(BuildMemifDirectory(mechanism), socketFilename)
}

func BuildMemifDirectory(mechanism *local.Mechanism) string {
	return path.Join(mechanism.Parameters[local.Workspace], MemifBaseDirectory)
}

func MasterSlave(src, dst *local.Mechanism) (*local.Mechanism, *local.Mechanism) {
	if isMaster, _ := strconv.ParseBool(src.GetParameters()[local.Master]); isMaster {
		return src, dst
	}
	return dst, src
}

func InvertMechanismRole(mechanism *local.Mechanism) {
	role := GetRole(mechanism)
	delete(mechanism.Parameters, role)
	mechanism.Parameters[invertRole(role)] = "true"
}

func GetRole(mechanism *local.Mechanism) string {
	if _, ok := mechanism.Parameters[local.Master]; ok {
		return local.Master
	}
	return local.Slave
}

func invertRole(role string) string {
	if role == local.Master {
		return local.Slave
	}
	return local.Master
}
