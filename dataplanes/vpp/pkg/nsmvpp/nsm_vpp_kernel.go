package nsmvpp

import (
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"

	govppapi "git.fd.io/govpp.git/api"
	"github.com/ligato/networkservicemesh/dataplanes/vpp/bin_api/tapv2"
	"github.com/ligato/networkservicemesh/dataplanes/vpp/pkg/nsmutils"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/common"
	"github.com/ligato/networkservicemesh/utils/fs"
	"github.com/sirupsen/logrus"
)

const lettersAndNumbers = "abcdefghijklmnopqrstuvwxyz0123456789"

func randString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = lettersAndNumbers[rand.Intn(len(lettersAndNumbers))]
	}
	return string(b)
}

type createLocalInterface struct {
	intf           *vppInterface
	localMechanism *common.LocalMechanism
}

type deleteLocalInterface struct {
	intf *vppInterface
}

func (c *createLocalInterface) apply(apiCh govppapi.Channel) error {
	switch c.localMechanism.Type {
	case common.LocalMechanismType_DEFAULT_INTERFACE,
		common.LocalMechanismType_KERNEL_INTERFACE:
		return createTapInterface(c, apiCh)
	case common.LocalMechanismType_MEM_INTERFACE:
		return memifInterfaceCreate(c, apiCh)
	default:
		return fmt.Errorf("create local interface for mechanism %d not implemented.", c.localMechanism.Type)
	}
}

func (op *createLocalInterface) rollback() operation {
	return &deleteLocalInterface{
		intf: op.intf,
	}
}

func createTapInterface(c *createLocalInterface, apiCh govppapi.Channel) error {
	logrus.Infof("Creating tap interface with parameters: %v...", c.localMechanism.Parameters)
	// Check presence of both ipv4 address and prefix length
	_, v1 := c.localMechanism.Parameters[nsmutils.NSMkeyIPv4]
	_, v2 := c.localMechanism.Parameters[nsmutils.NSMkeyIPv4PrefixLength]
	if v1 != v2 {
		return fmt.Errorf("both parameter \"ipv4\" and \"ipv4prefixlength\" must either present or missing")
	}

	err := nsmutils.ValidateParameters(c.localMechanism.Parameters, nsmutils.Keys{
		nsmutils.NSMkeyNamespace: nsmutils.KeyProperties{
			Mandatory: true,
			Validator: nsmutils.Any},
		nsmutils.NSMkeyIPv4: nsmutils.KeyProperties{
			Mandatory: false,
			Validator: nsmutils.Ipv4},
		nsmutils.NSMkeyIPv4PrefixLength: nsmutils.KeyProperties{
			Mandatory: false,
			Validator: nsmutils.Ipv4prefixlength},
	})
	if err != nil {
		return err
	}

	// Extract namespace
	namespace := c.localMechanism.Parameters[nsmutils.NSMkeyNamespace]

	if !strings.HasPrefix(namespace, "pid:") {
		// assuming that inode of linux namespace has been passed
		inode, err := strconv.ParseUint(namespace, 10, 64)
		if err != nil {
			logrus.Errorf("can't parse integer: %s", namespace)
		} else {
			namespace, err = fs.FindFileInProc(inode, "/ns/net")
			if err != nil {
				logrus.Errorf("cant' find namespace for inode %d", inode)
				return err
			}
		}
	}

	name := fmt.Sprintf("tap-%s", randString(6))
	ipv4, _ := c.localMechanism.Parameters[nsmutils.NSMkeyIPv4]

	ip, err := IPv4ToByteSlice(ipv4)
	if err != nil {
		return err
	}
	l, _ := strconv.Atoi(c.localMechanism.Parameters[nsmutils.NSMkeyIPv4PrefixLength])

	var tapReq tapv2.TapCreateV2
	var tapRpl tapv2.TapCreateV2Reply
	tapReq.ID = ^uint32(0)
	tapReq.HostNamespaceSet = 1
	tapReq.HostNamespace = []byte(namespace)
	tapReq.UseRandomMac = 1
	tapReq.Tag = []byte("NSM_CLIENT")
	tapReq.HostIfName = []byte(name)
	tapReq.HostIfNameSet = 1
	if len(ip) != 0 {
		tapReq.HostIP4Addr = ip
		tapReq.HostIP4PrefixLen = uint8(l)
		tapReq.HostIP4AddrSet = 1
	}
	if err := apiCh.SendRequest(&tapReq).ReceiveReply(&tapRpl); err != nil {
		return err
	}
	c.intf.id = tapRpl.SwIfIndex
	c.intf.mechanism = c.localMechanism

	logrus.Infof("created tap interface, name: %s", name)
	return nil
}

// deleteTapInterface creates new tap interface in a specified namespace
func deleteTapInterface(op *deleteLocalInterface, apiCh govppapi.Channel) error {
	logrus.Infof("Removing tap interface id %d...", op.intf.id)
	var tapReq tapv2.TapDeleteV2
	tapReq.SwIfIndex = op.intf.id
	if err := apiCh.SendRequest(&tapReq).ReceiveReply(&tapv2.TapDeleteV2Reply{}); err != nil {
		return err
	}
	return nil
}

func (op *deleteLocalInterface) apply(apiCh govppapi.Channel) error {
	switch op.intf.mechanism.Type {
	case common.LocalMechanismType_DEFAULT_INTERFACE,
		common.LocalMechanismType_KERNEL_INTERFACE:
		return deleteTapInterface(op, apiCh)
	case common.LocalMechanismType_MEM_INTERFACE:
		return memifInterfaceDelete(op, apiCh)
	default:
		return fmt.Errorf("delete local interface for mechanism %d not implemented.", op.intf.mechanism.Type)
	}
}

func (op *deleteLocalInterface) rollback() operation {
	return &createLocalInterface{
		intf:           op.intf,
		localMechanism: op.intf.mechanism,
	}
}

// IPv4ToByteSlice converts an ipv4 address in form '1.2.3.4' to an []byte]
// representation of the address.
func IPv4ToByteSlice(ipv4Address string) ([]byte, error) {
	var ipu []byte

	ipv4Address = strings.Trim(ipv4Address, " ")
	match, _ := regexp.Match(`^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$`,
		[]byte(ipv4Address))
	if !match {
		return nil, fmt.Errorf("invalid IP address %s", ipv4Address)
	}
	parts := strings.Split(ipv4Address, ".")
	for _, p := range parts {
		num, _ := strconv.Atoi(p)
		ipu = append(ipu, byte(num))
	}

	return ipu, nil
}
