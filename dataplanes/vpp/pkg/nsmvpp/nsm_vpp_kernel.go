package nsmvpp

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	govppapi "git.fd.io/govpp.git/api"
	"github.com/ligato/networkservicemesh/dataplanes/vpp/bin_api/tapv2"
	"github.com/ligato/networkservicemesh/dataplanes/vpp/pkg/nsmutils"
	"github.com/ligato/networkservicemesh/utils/fs"
	"github.com/sirupsen/logrus"
)

const lettersAndNumbers = "abcdefghijklmnopqrstuvwxyz0123456789"

type kernelMechanism struct{}

func randString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = lettersAndNumbers[rand.Intn(len(lettersAndNumbers))]
	}
	return string(b)
}

var keyList = nsmutils.Keys{
	nsmutils.NSMkeyNamespace: nsmutils.KeyProperties{
		Mandatory: true,
		Validator: nsmutils.Any},
	nsmutils.NSMkeyIPv4: nsmutils.KeyProperties{
		Mandatory: false,
		Validator: nsmutils.Ipv4},
	nsmutils.NSMkeyIPv4PrefixLength: nsmutils.KeyProperties{
		Mandatory: false,
		Validator: nsmutils.Ipv4prefixlength},
}

func (m kernelMechanism) validate(parameters map[string]string) error {
	// Check presence of both ipv4 address and prefix length
	_, v1 := parameters[nsmutils.NSMkeyIPv4]
	_, v2 := parameters[nsmutils.NSMkeyIPv4PrefixLength]
	if v1 != v2 {
		return fmt.Errorf("both parameter \"ipv4\" and \"ipv4prefixlength\" must either present or missing")
	}

	return nsmutils.ValidateParameters(parameters, keyList)
}

func (m kernelMechanism) createInterface(apiCh govppapi.Channel, parameters map[string]string) (uint32, error) {
	// Extract namespace
	namespace := parameters[nsmutils.NSMkeyNamespace]

	if !strings.HasPrefix(namespace, "pid:") {
		// assuming that inode of linux namespace has been passed
		inode, err := strconv.ParseUint(namespace, 10, 64)
		if err != nil {
			logrus.Errorf("can't parse integer: %s", namespace)
		} else {
			namespace, err = fs.FindFileInProc(inode, "/ns/net")
			if err != nil {
				logrus.Errorf("cant' find namespace for inode %d", inode)
				return 0, err
			}
		}
	}

	name := fmt.Sprintf("tap-%s", randString(6))
	ipv4, _ := parameters[nsmutils.NSMkeyIPv4]

	ip, err := IPv4ToByteSlice(ipv4)
	if err != nil {
		return 0, err
	}
	l, _ := strconv.Atoi(parameters[nsmutils.NSMkeyIPv4PrefixLength])

	logrus.Infof("Creating tap interface: %s", name)
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
		return 0, err
	}
	return tapRpl.SwIfIndex, nil
}

// deleteTapInterface creates new tap interface in a specified namespace
func (m kernelMechanism) deleteInterface(apiCh govppapi.Channel, tapIntfID uint32) error {
	var tapReq tapv2.TapDeleteV2
	tapReq.SwIfIndex = tapIntfID
	if err := apiCh.SendRequest(&tapReq).ReceiveReply(&tapv2.TapDeleteV2Reply{}); err != nil {
		return err
	}
	return nil
}
