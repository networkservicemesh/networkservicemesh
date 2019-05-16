package common

import (
	"bufio"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestParseDefaultGateway(t *testing.T) {
	RegisterTestingT(t)

	gw := parseGatewayIP("010011AC")
	logrus.Printf("Value %v", gw.String())
	Expect(gw.String(), "172.17.0.1")
}

func TestParseProcContent(t *testing.T) {
	RegisterTestingT(t)

	r := bufio.NewReader(strings.NewReader("Iface	Destination	Gateway 	Flags	RefCnt	Use	Metric	Mask		MTU	Window	IRTT\n" +
		"eth0	00000000	010011AC	0003	0	0	0	00000000	0	0	0\n" +
		"eth0	000011AC	00000000	0001	0	0	0	0000FFFF	0	0	0\n"))

	eth0, gw, err := parseProcFile(r)
	Expect(err).To(BeNil())
	logrus.Printf("Value %v", gw.String())
	Expect(gw.String()).To(Equal("172.17.0.1"))
	Expect(eth0).To(Equal("eth0"))
}
func TestParseProcWrongContent(t *testing.T) {
	RegisterTestingT(t)

	r := bufio.NewReader(strings.NewReader("Iface	Destination	Gateway 	Flags	RefCnt	Use	Metric	Mask		MTU	Window	IRTT\n" +
		"eth0	00000001	010011AC	0003	0	0	0	00000000	0	0	0\n" +
		"eth0	000011AC	00000000	0001	0	0	0	0000FFFF	0	0	0\n"))

	eth0, gw, err := parseProcFile(r)
	Expect(err.Error()).To(Equal("Failed to locate default route..."))
	logrus.Printf("Value %v", gw.String())
	Expect(eth0).To(Equal(""))
	Expect(gw).To(BeNil())
}
