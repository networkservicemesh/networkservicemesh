package local

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/common"
	"github.com/networkservicemesh/networkservicemesh/utils/fs"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// CreateRemoteInterface - creates interface to remote connection
func CreateLocalInterface(ifaceName string, conn *connection.Connection) (string, error) {
	netNsInode := conn.GetMechanism().GetParameters()[common.NetNsInodeKey]
	ifaceIP := conn.GetContext().GetIpContext().GetSrcIpAddr()
	routes := conn.GetContext().GetIpContext().GetDstRoutes()
	neighbors := conn.GetContext().GetIpContext().GetIpNeighbors()

	/* Get namespace handler - source */
	nsHandle, err := fs.GetNsHandleFromInode(netNsInode)
	if err != nil {
		logrus.Errorf("local: failed to get source namespace handle - %v", err)
		return netNsInode, err
	}
	/* If successful, don't forget to close the handler upon exit */
	defer func() {
		if err = nsHandle.Close(); err != nil {
			logrus.Error("local: error when closing source handle: ", err)
		}
		logrus.Debug("local: closed source handle: ", nsHandle, netNsInode)
	}()
	logrus.Debug("local: opened source handle: ", nsHandle, netNsInode)


	/* Setup interface - source namespace */
	if err = setupLinkInNs(nsHandle, ifaceName, ifaceIP, routes, neighbors, true); err != nil {
		logrus.Errorf("local: failed to setup interface - source - %q: %v", ifaceName, err)
		return netNsInode, err
	}

	return netNsInode, nil
}

// CreateRemoteInterface - deletes interface to remote connection
func DeleteLocalInterface(ifaceName string, conn *connection.Connection) (string, error) {
	netNsInode := conn.GetMechanism().GetParameters()[common.NetNsInodeKey]
	ifaceIP := conn.GetContext().GetIpContext().GetSrcIpAddr()

	/* Get namespace handler - source */
	nsHandle, err := fs.GetNsHandleFromInode(netNsInode)
	if err != nil {
		return "", errors.Errorf("failed to get source namespace handle - %v", err)
	}
	/* If successful, don't forget to close the handler upon exit */
	defer func() {
		if err = nsHandle.Close(); err != nil {
			logrus.Error("local: error when closing source handle: ", err)
		}
		logrus.Debug("local: closed source handle: ", nsHandle, netNsInode)
	}()
	logrus.Debug("local: opened source handle: ", nsHandle, netNsInode)

	/* Extract interface - source namespace */
	if err = setupLinkInNs(nsHandle, ifaceName, ifaceIP, nil, nil, false); err != nil {
		return "", errors.Errorf("failed to extract interface - source - %q: %v", ifaceName, err)

	}

	return netNsInode, nil
}
