package common

import (
	"bufio"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)
func TestParseGatewayZero(t *testing.T) {
	g := NewWithT(t)

	gw, err := parseGatewayIP("0")
	g.Expect(err.Error()).To(Equal("Failed to locate default route..."))
	g.Expect(gw.IsUnspecified()).To(BeTrue())
}
func TestParseDefaultGateway(t *testing.T) {
	g := NewWithT(t)

	gw, err := parseGatewayIP("010011AC")
	g.Expect(err).To(BeNil())
	logrus.Printf("Value %v", gw.String())
	g.Expect(gw.String(), "172.17.0.1")
}

func TestParseProcContent(t *testing.T) {
	g := NewWithT(t)

	r := bufio.NewReader(strings.NewReader("Iface	Destination	Gateway 	Flags	RefCnt	Use	Metric	Mask		MTU	Window	IRTT\n" +
		"eth0	00000000	010011AC	0003	0	0	0	00000000	0	0	0\n" +
		"eth0	000011AC	00000000	0001	0	0	0	0000FFFF	0	0	0\n"))

	eth0, gw, err := parseProcFile(r)
	g.Expect(err).To(BeNil())
	logrus.Printf("Value %v", gw.String())
	g.Expect(gw.String()).To(Equal("172.17.0.1"))
	g.Expect(eth0).To(Equal("eth0"))
}
func TestParseProcWrongContent(t *testing.T) {
	g := NewWithT(t)

	r := bufio.NewReader(strings.NewReader("Iface	Destination	Gateway 	Flags	RefCnt	Use	Metric	Mask		MTU	Window	IRTT\n" +
		"eth0	00000001	010011AC	0003	0	0	0	00000000	0	0	0\n" +
		"eth0	000011AC	00000000	0001	0	0	0	0000FFFF	0	0	0\n"))

	eth0, gw, err := parseProcFile(r)
	g.Expect(err.Error()).To(Equal("Failed to locate default route..."))
	logrus.Printf("Value %v", gw.String())
	g.Expect(eth0).To(Equal(""))
	g.Expect(gw).To(BeNil())
}
