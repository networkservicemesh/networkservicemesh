package common

import (
	"bufio"
	"bytes"
	"strings"
	"testing"

	"github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestParseGatewayLessThenEight(t *testing.T) {
	g := gomega.NewWithT(t)

	gw, err := parseGatewayIP("0")
	g.Expect(err.Error()).To(gomega.Equal("failed to parse IP from string"))
	g.Expect(gw).To(gomega.BeNil())
}

func TestParseGatewayStringLengthGreaterThenEight(t *testing.T) {
	g := gomega.NewWithT(t)

	gw, err := parseGatewayIP("111111111")
	g.Expect(err.Error()).To(gomega.Equal("failed to parse IP from string"))
	g.Expect(gw).To(gomega.BeNil())
}

func TestParseDefaultGatewayValidString(t *testing.T) {
	g := gomega.NewWithT(t)

	gw, err := parseGatewayIP("010011AC")
	g.Expect(err).To(gomega.BeNil())
	logrus.Printf("Value %v", gw.String())
	g.Expect(gw.String()).To(gomega.Equal("172.17.0.1"))
}

func TestParseDefaultGatewayInvalidString(t *testing.T) {
	g := gomega.NewWithT(t)

	gw, err := parseGatewayIP("010011AS")
	g.Expect(err.Error()).To(gomega.Equal("string does not represent a valid IP address"))
	logrus.Printf("Value %v", gw.String())
	g.Expect(gw).To(gomega.BeNil())
}

func TestParseProcBlankLine(t *testing.T) {
	g := gomega.NewWithT(t)

	s := bufio.NewScanner(bytes.NewBuffer([]byte{0x0a}))
	eth0, gw, err := parseProcFile(s)
	g.Expect(err.Error()).To(gomega.Not(gomega.BeNil()))
	g.Expect(eth0).To(gomega.Equal(""))
	g.Expect(gw).To(gomega.BeNil())
}

func TestParseProcContent(t *testing.T) {
	g := gomega.NewWithT(t)

	s := bufio.NewScanner(strings.NewReader("Iface	Destination	Gateway 	Flags	RefCnt	Use	Metric	Mask		MTU	Window	IRTT\n" +
		"eth0	00000000	010011AC	0003	0	0	0	00000000	0	0	0\n" +
		"eth0	000011AC	00000000	0001	0	0	0	0000FFFF	0	0	0\n" +
		"\n" +
		"eth1	000011AB	00000000	0004	0	0	0	0000BBBB	0	0	0\n"))

	eth0, gw, err := parseProcFile(s)
	g.Expect(err).To(gomega.BeNil())
	logrus.Printf("Value %v", gw.String())
	g.Expect(gw.String()).To(gomega.Equal("172.17.0.1"))
	g.Expect(eth0).To(gomega.Equal("eth0"))
}
func TestParseProcWrongContent(t *testing.T) {
	g := gomega.NewWithT(t)

	s := bufio.NewScanner(strings.NewReader("Iface	Destination	Gateway 	Flags	RefCnt	Use	Metric	Mask		MTU	Window	IRTT\n" +
		"eth0	00000001	010011AC	0003	0	0	0	00000000	0	0	0\n" +
		"eth0	000011AC	00000000	0001	0	0	0	0000FFFF	0	0	0\n"))

	eth0, gw, err := parseProcFile(s)
	g.Expect(err.Error()).To(gomega.Equal("failed to locate default route..."))
	logrus.Printf("Value %v", gw.String())
	g.Expect(eth0).To(gomega.Equal(""))
	g.Expect(gw).To(gomega.BeNil())
}
