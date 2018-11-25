package memif

import (
	"fmt"
	"github.com/docker/docker/pkg/mount"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/crossconnect"
	local "github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/sirupsen/logrus"
	"os"
	"path"
)

func DirectConnection(crossConnect *crossconnect.CrossConnect) (*crossconnect.CrossConnect, error) {
	srcMechanism := crossConnect.GetLocalSource().GetMechanism()
	dstMechanism := crossConnect.GetLocalDestination().GetMechanism()
	master, slave := MasterSlave(srcMechanism, dstMechanism)

	masterSocketDir := path.Join(BuildMemifDirectory(master), crossConnect.Id)
	slaveSocketDir := path.Join(BuildMemifDirectory(slave), crossConnect.Id)

	if err := CreateDirectory(masterSocketDir); err != nil {
		return nil, err
	}

	if err := CreateDirectory(slaveSocketDir); err != nil {
		return nil, err
	}

	if err := mount.Mount(masterSocketDir, slaveSocketDir, "hard", "bind"); err != nil {
		return nil, err
	}
	logrus.Infof("Successfully mount folder %s to %s", masterSocketDir, slaveSocketDir)

	if master.GetParameters()[local.SocketFilename] != slave.GetParameters()[local.SocketFilename] {
		masterSocket := path.Join(masterSocketDir, master.GetParameters()[local.SocketFilename])
		slaveSocket := path.Join(slaveSocketDir, slave.GetParameters()[local.SocketFilename])

		if err := os.Symlink(masterSocket, slaveSocket); err != nil {
			return nil, fmt.Errorf("failed to create symlink: %s", err)
		}
	}

	return crossConnect, nil
}
