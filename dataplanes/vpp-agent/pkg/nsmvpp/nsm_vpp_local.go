package nsmvpp

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/ligato/networkservicemesh/dataplanes/vpp-agent/pkg/nsmutils"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/common"
	"github.com/ligato/networkservicemesh/utils/fs"
	"github.com/ligato/vpp-agent/clientv1/vpp/remoteclient"
	"github.com/ligato/vpp-agent/plugins/vpp/model/interfaces"
	"github.com/ligato/vpp-agent/plugins/vpp/model/rpc"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type vppInterface struct {
	mechanism *common.LocalMechanism // we want to save parameters here in order to recreate interface
	id        uint32
}

var (
	connections map[int][]operation = make(map[int][]operation)
	lastId      int                 = 0
)

const (
	address = "localhost:9111"
)

func createTapInterface(name, namespace, ipAddress string) *interfaces.Interfaces_Interface {
	return &interfaces.Interfaces_Interface{
		Name:        name,
		Type:        interfaces.InterfaceType_TAP_INTERFACE,
		Enabled:     true,
		IpAddresses: []string{ipAddress},
		Tap: &interfaces.Interfaces_Interface_Tap{
			Version:    2,
			HostIfName: "linux-" + name,
			Namespace:  namespace,
		},
		Mtu: 1500,
	}
}

func fixNamespace(namespace string) (string, error) {
	// Extract namespace
	if !strings.HasPrefix(namespace, "pid:") {
		// assuming that inode of linux namespace has been passed
		inode, err := strconv.ParseUint(namespace, 10, 64)
		if err != nil {
			logrus.Errorf("can't parse integer: %s", namespace)
		} else {
			namespace, err = fs.FindFileInProc(inode, "/ns/net")
			if err != nil {
				logrus.Errorf("cant' find namespace for inode %d", inode)
				return "", err
			}
		}
	}
	return namespace, nil
}

// CreateLocalConnect sanity checks parameters passed in the LocalMechanisms and call nsmvpp.CreateLocalConnect
func CreateLocalConnect(client *VPPAgentClient, src, dst *common.LocalMechanism) (string, error) {
	logrus.Infof("L O C A L  C O N N E C T")

	conn, err := grpc.Dial(":9111", grpc.WithInsecure())
	if err != nil {
		logrus.Errorf("can't dial grpc server: %v", err)
	}
	defer conn.Close()

	srcNamespace, err := fixNamespace(src.Parameters[nsmutils.NSMkeyNamespace])
	if err != nil {
		logrus.Errorf("namespace error: %v", err)
	}

	srcIpAddress := fmt.Sprintf("%s/%s", src.Parameters[nsmutils.NSMkeyIPv4], src.Parameters[nsmutils.NSMkeyIPv4PrefixLength])
	logrus.Infof("src namespace %s, ip addess %s", srcNamespace, srcIpAddress)

	dstNamespace, err := fixNamespace(dst.Parameters[nsmutils.NSMkeyNamespace])
	if err != nil {
		logrus.Errorf("namespace error: %v", err)
	}

	dstIpAddress := fmt.Sprintf("%s/%s", dst.Parameters[nsmutils.NSMkeyIPv4], dst.Parameters[nsmutils.NSMkeyIPv4PrefixLength])
	logrus.Infof("dst namespace %s, ip addess %s", dstNamespace, dstIpAddress)

	err = remoteclient.DataResyncRequestGRPC(rpc.NewDataResyncServiceClient(conn)).
		Interface(createTapInterface("tap1", srcNamespace, srcIpAddress)).
		Interface(createTapInterface("tap2", dstNamespace, dstIpAddress)).
		Send().ReceiveReply()
	if err != nil {
		logrus.Errorf("Failed to apply initial VPP configuration: %v", err)
	} else {
		logrus.Info("Successfully applied initial VPP configuration")
	}

	return fmt.Sprintf("%d", lastId), nil
}

// Dialer for unix domain socket
func dialer(socket, address string, timeoutVal time.Duration) func(string, time.Duration) (net.Conn, error) {
	return func(addr string, timeout time.Duration) (net.Conn, error) {
		// Pass values
		addr, timeout = address, timeoutVal
		// Dial with timeout
		return net.DialTimeout(socket, addr, timeoutVal)
	}
}

// DeleteLocalConnect
func DeleteLocalConnect(client *VPPAgentClient, connID string) error {
	// id, _ := strconv.Atoi(connID)
	// tx := connections[id]
	return nil
}
