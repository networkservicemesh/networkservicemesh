package converter

import (
	"fmt"
	"net"
	"os"

	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
)

const (
	ForwarderAllowVHost = "FORWARDER_ALLOW_VHOST" // To disallow VHOST please pass "false" into this env variable.
	dstInterfaceFormat  = "DST-%v"
	srcInterfaceFormat  = "SRC-%v"
)

func TempIfName() string {
	// xids are  12 bytes -
	// 4-byte value representing the seconds since the Unix epoch,
	// 3-byte machine identifier,
	// 2-byte process id, and
	// 3-byte counter, starting with a random value.
	guid := xid.New()

	// We need something randomish but not more than 15 bytes
	// Obviously we only care about the first 4 bytes and the last 3 bytes
	// xid encodes to base32, so each char represents 5 bits
	// 4*8/5 =  6.4 - so if we grab the first 7 chars thats going to include
	// the first four bytes
	// 3*8/5 = 4.8 - so if we grab the last 5 chars that will include the
	// last three bytes

	rv := guid.String()
	rv = rv[:7] + rv[16:]
	logrus.Infof("Generated unique TempIfName: %s len(TempIfName) %d", rv, len(rv))
	return rv
}

//GetDstInterfaceName returns name of dst interface by id
func GetDstInterfaceName(id string) string {
	return fmt.Sprintf(dstInterfaceFormat, id)
}

//GetSrcInterfaceName returns name of src interface by id
func GetSrcInterfaceName(id string) string {
	return fmt.Sprintf(srcInterfaceFormat, id)
}

func useVHostNet() bool {
	vhostAllowed := os.Getenv(ForwarderAllowVHost)
	if vhostAllowed == "false" {
		return false
	}
	if _, err := os.Stat("/dev/vhost-net"); err == nil {
		return true
	}
	return false
}

func extractCleanIPAddress(addr string) string {
	ip, _, err := net.ParseCIDR(addr)
	if err == nil {
		return ip.String()
	}
	return addr
}
