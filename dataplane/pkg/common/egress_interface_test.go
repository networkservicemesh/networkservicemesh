package common

import (
	"bufio"
	"bytes"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestParseGatewayLessThenEight(t *testing.T) {
	g := NewWithT(t)

	gw, err := parseGatewayIP("0")
	g.Expect(err.Error()).To(Equal("Failed to parse IP from string"))
	g.Expect(gw.IsUnspecified()).To(BeTrue())
}

func TestParseGatewayStringLengthGreaterThenEight(t *testing.T) {
	g := NewWithT(t)

	gw, err := parseGatewayIP("111111111")
	g.Expect(err.Error()).To(Equal("Failed to parse IP from string"))
	g.Expect(gw.IsUnspecified()).To(BeTrue())
}

func TestParseDefaultGatewayValidString(t *testing.T) {
	g := NewWithT(t)

	gw, err := parseGatewayIP("010011AC")
	g.Expect(err).To(BeNil())
	logrus.Printf("Value %v", gw.String())
	g.Expect(gw.String()).To(Equal("172.17.0.1"))
}

func TestParseDefaultGatewayInvalidIPString(t *testing.T) {
	g := NewWithT(t)

	gw, err := parseGatewayIP("010011AS")
	g.Expect(err).To(BeNil())
	logrus.Printf("Value %v", gw.String())
	g.Expect(gw.String()).To(Equal("0.17.0.2"))
}

func TestParseProcBlankLine(t *testing.T) {
	g := NewWithT(t)

	s := bufio.NewScanner(bytes.NewBuffer([]byte{0x0a}))
	eth0, gw, err := parseProcFile(s)
	g.Expect(err.Error()).To(Not(BeNil()))
	g.Expect(eth0).To(Equal(""))
	g.Expect(gw).To(BeNil())
}

func TestParseProcContent(t *testing.T) {
	g := NewWithT(t)

	s := bufio.NewScanner(strings.NewReader("Iface	Destination	Gateway 	Flags	RefCnt	Use	Metric	Mask		MTU	Window	IRTT\n" +
		"eth0	00000000	010011AC	0003	0	0	0	00000000	0	0	0\n" +
		"eth0	000011AC	00000000	0001	0	0	0	0000FFFF	0	0	0\n" +
		"\n" +
		"eth1	000011AB	00000000	0004	0	0	0	0000BBBB	0	0	0\n"))

	eth0, gw, err := parseProcFile(s)
	g.Expect(err).To(BeNil())
	logrus.Printf("Value %v", gw.String())
	g.Expect(gw.String()).To(Equal("172.17.0.1"))
	g.Expect(eth0).To(Equal("eth0"))
}
func TestParseProcWrongContent(t *testing.T) {
	g := NewWithT(t)

	s := bufio.NewScanner(strings.NewReader("Iface	Destination	Gateway 	Flags	RefCnt	Use	Metric	Mask		MTU	Window	IRTT\n" +
		"eth0	00000001	010011AC	0003	0	0	0	00000000	0	0	0\n" +
		"eth0	000011AC	00000000	0001	0	0	0	0000FFFF	0	0	0\n"))

	eth0, gw, err := parseProcFile(s)
	g.Expect(err.Error()).To(Equal("Failed to locate default route..."))
	logrus.Printf("Value %v", gw.String())
	g.Expect(eth0).To(Equal(""))
	g.Expect(gw).To(BeNil())
}
