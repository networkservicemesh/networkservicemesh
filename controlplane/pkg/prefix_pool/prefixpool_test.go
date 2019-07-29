package prefix_pool

import (
	"fmt"
	"net"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
)

func TestPrefixPoolSubnet1(t *testing.T) {
	g := NewWithT(t)

	prefixes := []string{"10.10.1.0/24"}
	logrus.Printf("Address count: %d", AddressCount(prefixes...))
	g.Expect(AddressCount(prefixes...)).To(Equal(uint64(256)))

	_, snet1, _ := net.ParseCIDR("10.10.1.0/24")
	sn1, err := subnet(snet1, 0)
	g.Expect(err).To(BeNil())
	logrus.Printf(sn1.String())
	g.Expect(sn1.String()).To(Equal("10.10.1.0/25"))
	s, e := AddressRange(sn1)
	g.Expect(s.String()).To(Equal("10.10.1.0"))
	g.Expect(e.String()).To(Equal("10.10.1.127"))
	g.Expect(addressCount(sn1.String())).To(Equal(uint64(128)))

	lastIp := s
	for i := uint64(0); i < addressCount((sn1.String()))-1; i++ {
		ip, err := IncrementIP(lastIp, sn1)
		g.Expect(err).To(BeNil())
		lastIp = ip
	}

	_, err = IncrementIP(lastIp, sn1)
	g.Expect(err.Error()).To(Equal("Overflowed CIDR while incrementing IP"))

	sn2, err := subnet(snet1, 1)
	g.Expect(err).To(BeNil())
	logrus.Printf(sn2.String())
	g.Expect(sn2.String()).To(Equal("10.10.1.128/25"))
	s, e = AddressRange(sn2)
	g.Expect(s.String()).To(Equal("10.10.1.128"))
	g.Expect(e.String()).To(Equal("10.10.1.255"))
	g.Expect(addressCount(sn2.String())).To(Equal(uint64(128)))
}

func TestNetExtractIPv4(t *testing.T) {
	testNetExtract(t, "10.10.1.0/24", "10.10.1.1/30", "10.10.1.2/30", connectioncontext.IpFamily_IPV4)
}

func TestNetExtractIPv6(t *testing.T) {
	testNetExtract(t, "100::/64", "100::1/126", "100::2/126", connectioncontext.IpFamily_IPV6)
}

func testNetExtract(t *testing.T, inPool, srcDesired, dstDesired string, family connectioncontext.IpFamily_Family) {
	g := NewWithT(t)

	pool, err := NewPrefixPool(inPool)
	g.Expect(err).To(BeNil())

	srcIP, dstIP, requested, err := pool.Extract("c1", family)
	g.Expect(err).To(BeNil())
	g.Expect(requested).To(BeNil())

	g.Expect(srcIP.String()).To(Equal(srcDesired))
	g.Expect(dstIP.String()).To(Equal(dstDesired))

	err = pool.Release("c1")
	g.Expect(err).To(BeNil())
}

func TestExtract1(t *testing.T) {
	g := NewWithT(t)

	newPrefixes, err := ReleasePrefixes([]string{"10.10.1.0/25"}, "10.10.1.127/25")
	g.Expect(err).To(BeNil())
	g.Expect(newPrefixes).To(Equal([]string{"10.10.1.0/24"}))
	logrus.Printf("%v", newPrefixes)
}

func TestExtractPrefixes_1_ipv4(t *testing.T) {
	g := NewWithT(t)

	newPrefixes, prefixes, err := ExtractPrefixes([]string{"10.10.1.0/24"},
		&connectioncontext.ExtraPrefixRequest{
			AddrFamily:      &connectioncontext.IpFamily{Family: connectioncontext.IpFamily_IPV4},
			RequiredNumber:  10,
			RequestedNumber: 20,
			PrefixLen:       31,
		},
	)
	g.Expect(err).To(BeNil())
	g.Expect(len(newPrefixes)).To(Equal(20))
	g.Expect(len(prefixes)).To(Equal(4))
	logrus.Printf("%v", newPrefixes)
}

func TestExtractPrefixes_1_ipv6(t *testing.T) {
	g := NewWithT(t)

	newPrefixes, prefixes, err := ExtractPrefixes([]string{"100::/64"},
		&connectioncontext.ExtraPrefixRequest{
			AddrFamily:      &connectioncontext.IpFamily{Family: connectioncontext.IpFamily_IPV6},
			RequiredNumber:  100,
			RequestedNumber: 200,
			PrefixLen:       128,
		},
	)
	g.Expect(err).To(BeNil())
	g.Expect(len(newPrefixes)).To(Equal(200))
	g.Expect(len(prefixes)).To(Equal(59))
	logrus.Printf("%v", newPrefixes)
}

func TestExtract2(t *testing.T) {
	g := NewWithT(t)

	prefix, prefixes, err := ExtractPrefix([]string{"10.10.1.0/24"}, 24)
	g.Expect(err).To(BeNil())
	g.Expect(prefix).To(Equal("10.10.1.0/24"))
	g.Expect(len(prefixes)).To(Equal(0))
}

func TestExtract2_ipv6(t *testing.T) {
	g := NewWithT(t)

	prefix, prefixes, err := ExtractPrefix([]string{"100::/64"}, 65)
	g.Expect(err).To(BeNil())
	g.Expect(prefix).To(Equal("100::/65"))
	g.Expect(len(prefixes)).To(Equal(1))
	g.Expect(prefixes[0]).To(Equal("100::8000:0:0:0/65"))
}

func TestExtract3_ipv6(t *testing.T) {
	g := NewWithT(t)

	prefix, prefixes, err := ExtractPrefix([]string{"100::/64"}, 128)
	g.Expect(err).To(BeNil())
	g.Expect(prefix).To(Equal("100::/128"))
	g.Expect(len(prefixes)).To(Equal(64))
}

func TestRelease_ipv6(t *testing.T) {
	g := NewWithT(t)

	prefixes := []string{
		"100::1/128",
		"100::2/127",
		"100::4/126",
		"100::8/125",
		"100::10/124",
		"100::20/123",
		"100::40/122",
		"100::80/121",
		"100::100/120",
		"100::200/119",
		"100::400/118",
		"100::800/117",
		"100::1000/116",
		"100::2000/115",
		"100::4000/114",
		"100::8000/113",
		"100::1:0/112",
		"100::2:0/111",
		"100::4:0/110",
		"100::8:0/109",
		"100::10:0/108",
		"100::20:0/107",
		"100::40:0/106",
		"100::80:0/105",
		"100::100:0/104",
		"100::200:0/103",
		"100::400:0/102",
		"100::800:0/101",
		"100::1000:0/100",
		"100::2000:0/99",
		"100::4000:0/98",
		"100::8000:0/97",
		"100::1:0:0/96",
		"100::2:0:0/95",
		"100::4:0:0/94",
		"100::8:0:0/93",
		"100::10:0:0/92",
		"100::20:0:0/91",
		"100::40:0:0/90",
		"100::80:0:0/89",
		"100::100:0:0/88",
		"100::200:0:0/87",
		"100::400:0:0/86",
		"100::800:0:0/85",
		"100::1000:0:0/84",
		"100::2000:0:0/83",
		"100::4000:0:0/82",
		"100::8000:0:0/81",
		"100::1:0:0:0/80",
		"100::2:0:0:0/79",
		"100::4:0:0:0/78",
		"100::8:0:0:0/77",
		"100::10:0:0:0/76",
		"100::20:0:0:0/75",
		"100::40:0:0:0/74",
		"100::80:0:0:0/73",
		"100::100:0:0:0/72",
		"100::200:0:0:0/71",
		"100::400:0:0:0/70",
		"100::800:0:0:0/69",
		"100::1000:0:0:0/68",
		"100::2000:0:0:0/67",
		"100::4000:0:0:0/66",
		"100::8000:0:0:0/65",
	}
	released, err := ReleasePrefixes(prefixes, "100::/128")
	g.Expect(err).To(BeNil())
	g.Expect(len(released)).To(Equal(1))
	g.Expect(released[0]).To(Equal("100::/64"))
}

func TestExtract3(t *testing.T) {
	g := NewWithT(t)

	prefix, prefixes, err := ExtractPrefix([]string{"10.10.1.0/24"}, 23)
	g.Expect(err.Error()).To(Equal("Failed to find room to have prefix len 23 at [10.10.1.0/24]"))
	g.Expect(prefix).To(Equal(""))
	g.Expect(len(prefixes)).To(Equal(1))
}

func TestExtract4(t *testing.T) {
	g := NewWithT(t)

	prefix, prefixes, err := ExtractPrefix([]string{"10.10.1.0/24"}, 25)
	g.Expect(err).To(BeNil())
	g.Expect(prefix).To(Equal("10.10.1.0/25"))
	g.Expect(prefixes).To(Equal([]string{"10.10.1.128/25"}))
}

func TestExtract5(t *testing.T) {
	g := NewWithT(t)

	prefix, prefixes, err := ExtractPrefix([]string{"10.10.1.0/24"}, 26)
	g.Expect(err).To(BeNil())
	g.Expect(prefix).To(Equal("10.10.1.0/26"))
	g.Expect(prefixes).To(Equal([]string{"10.10.1.64/26", "10.10.1.128/25"}))
}

func TestExtract6(t *testing.T) {
	g := NewWithT(t)

	prefix, prefixes, err := ExtractPrefix([]string{"10.10.1.0/24"}, 32)
	g.Expect(err).To(BeNil())
	g.Expect(prefix).To(Equal("10.10.1.0/32"))
	g.Expect(prefixes).To(Equal([]string{"10.10.1.1/32", "10.10.1.2/31", "10.10.1.4/30", "10.10.1.8/29", "10.10.1.16/28", "10.10.1.32/27", "10.10.1.64/26", "10.10.1.128/25"}))
}

func TestExtract7(t *testing.T) {
	g := NewWithT(t)

	prefix, prefixes, err := ExtractPrefix([]string{"10.10.1.1/32", "10.10.1.2/31", "10.10.1.4/30", "10.10.1.8/29", "10.10.1.16/28", "10.10.1.32/27", "10.10.1.64/26", "10.10.1.128/25"}, 31)
	g.Expect(err).To(BeNil())
	g.Expect(prefix).To(Equal("10.10.1.2/31"))
	g.Expect(prefixes).To(Equal([]string{"10.10.1.1/32", "10.10.1.4/30", "10.10.1.8/29", "10.10.1.16/28", "10.10.1.32/27", "10.10.1.64/26", "10.10.1.128/25"}))
}
func TestExtract8(t *testing.T) {
	g := NewWithT(t)

	prefix, prefixes, err := ExtractPrefix([]string{"10.10.1.128/25", "10.10.1.2/31", "10.10.1.4/30", "10.10.1.8/29", "10.10.1.16/28", "10.10.1.32/27", "10.10.1.64/26"}, 32)
	g.Expect(err).To(BeNil())
	g.Expect(prefix).To(Equal("10.10.1.2/32"))
	g.Expect(prefixes).To(Equal([]string{"10.10.1.128/25", "10.10.1.3/32", "10.10.1.4/30", "10.10.1.8/29", "10.10.1.16/28", "10.10.1.32/27", "10.10.1.64/26"}))
}

func TestRelease1(t *testing.T) {
	g := NewWithT(t)

	newPrefixes, err := ReleasePrefixes([]string{"10.10.1.0/25"}, "10.10.1.127/25")
	g.Expect(err).To(BeNil())
	g.Expect(newPrefixes).To(Equal([]string{"10.10.1.0/24"}))
}

func TestRelease2(t *testing.T) {
	g := NewWithT(t)

	_, snet, _ := net.ParseCIDR("10.10.1.0/25")
	sn1, err := subnet(snet, 0)
	sn2, err := subnet(snet, 1)
	logrus.Printf("%v %v", sn1.String(), sn2.String())

	sn10 := clearNetIndexInIP(sn1.IP, 26)
	sn11 := clearNetIndexInIP(sn1.IP, 26)
	logrus.Printf("%v %v", sn10.String(), sn11.String())

	sn20 := clearNetIndexInIP(sn2.IP, 26)
	sn21 := clearNetIndexInIP(sn2.IP, 26)
	logrus.Printf("%v %v", sn20.String(), sn21.String())

	newPrefixes, err := ReleasePrefixes([]string{"10.10.1.64/26", "10.10.1.128/25"}, "10.10.1.0/26")
	g.Expect(err).To(BeNil())
	g.Expect(newPrefixes).To(Equal([]string{"10.10.1.0/24"}))
}

func TestIntersect1(t *testing.T) {
	g := NewWithT(t)

	pp, err := NewPrefixPool("10.10.1.0/24")
	g.Expect(err).To(BeNil())

	g.Expect(pp.Intersect("10.10.1.0/28")).To(Equal(true))
	g.Expect(pp.Intersect("10.10.1.10/28")).To(Equal(true))
	g.Expect(pp.Intersect("10.10.1.0/10")).To(Equal(true))
	g.Expect(pp.Intersect("10.10.0.0/10")).To(Equal(true))
	g.Expect(pp.Intersect("10.10.0.0/24")).To(Equal(false))
	g.Expect(pp.Intersect("10.10.1.0/24")).To(Equal(true))
}

func TestIntersect2(t *testing.T) {
	g := NewWithT(t)

	pp, err := NewPrefixPool("10.10.1.0/24", "10.32.1.0/16")
	g.Expect(err).To(BeNil())

	g.Expect(pp.Intersect("10.10.1.0/28")).To(Equal(true))
	g.Expect(pp.Intersect("10.10.1.10/28")).To(Equal(true))
	g.Expect(pp.Intersect("10.10.1.0/10")).To(Equal(true))
	g.Expect(pp.Intersect("10.10.0.0/10")).To(Equal(true))

	g.Expect(pp.Intersect("10.32.0.0/10")).To(Equal(true))
	g.Expect(pp.Intersect("10.32.0.0/24")).To(Equal(true))
	g.Expect(pp.Intersect("10.2.0.0/16")).To(Equal(false))

}

func TestReleaseExcludePrefixes(t *testing.T) {
	g := NewWithT(t)

	pool, err := NewPrefixPool("10.20.0.0/16")
	g.Expect(err).To(BeNil())
	excludedPrefix := []string{"10.20.1.10/24", "10.20.32.0/19"}

	excluded, err := pool.ExcludePrefixes(excludedPrefix)

	g.Expect(err).To(BeNil())
	g.Expect(pool.GetPrefixes()).To(Equal([]string{"10.20.0.0/24", "10.20.128.0/17", "10.20.64.0/18", "10.20.16.0/20", "10.20.8.0/21", "10.20.4.0/22", "10.20.2.0/23"}))

	err = pool.ReleaseExcludedPrefixes(excluded)
	g.Expect(err).To(BeNil())
	g.Expect(pool.GetPrefixes()).To(Equal([]string{"10.20.0.0/16"}))
}

func TestReleaseExcludePrefixesNoOverlap(t *testing.T) {
	g := NewWithT(t)

	pool, err := NewPrefixPool("10.20.0.0/16")
	g.Expect(err).To(BeNil())
	excludedPrefix := []string{"10.32.0.0/16"}

	excluded, err := pool.ExcludePrefixes(excludedPrefix)

	g.Expect(err).To(BeNil())
	g.Expect(pool.GetPrefixes()).To(Equal([]string{"10.20.0.0/16"}))

	err = pool.ReleaseExcludedPrefixes(excluded)
	g.Expect(err).To(BeNil())
	g.Expect(pool.GetPrefixes()).To(Equal([]string{"10.20.0.0/16"}))
}

func TestReleaseExcludePrefixesFullOverlap(t *testing.T) {
	g := NewWithT(t)
	pool, err := NewPrefixPool("10.20.0.0/16", "2.20.0.0/16")
	g.Expect(err).To(BeNil())
	excludedPrefix := []string{"2.20.0.0/8"}

	excluded, err := pool.ExcludePrefixes(excludedPrefix)

	g.Expect(err).To(BeNil())
	g.Expect(pool.GetPrefixes()).To(Equal([]string{"10.20.0.0/16"}))

	err = pool.ReleaseExcludedPrefixes(excluded)
	g.Expect(err).To(BeNil())
	g.Expect(pool.GetPrefixes()).To(Equal([]string{"10.20.0.0/16", "2.20.0.0/16"}))
}

func TestExcludePrefixesPartialOverlap(t *testing.T) {
	g := NewWithT(t)

	pool, err := NewPrefixPool("10.20.0.0/16", "10.32.0.0/16")
	g.Expect(err).To(BeNil())
	excludedPrefix := []string{"10.20.1.10/24", "10.20.32.0/19"}

	_, err = pool.ExcludePrefixes(excludedPrefix)

	g.Expect(err).To(BeNil())
	g.Expect(pool.GetPrefixes()).To(Equal([]string{"10.20.0.0/24", "10.20.128.0/17", "10.20.64.0/18", "10.20.16.0/20", "10.20.8.0/21", "10.20.4.0/22", "10.20.2.0/23", "10.32.0.0/16"}))
}

func TestExcludePrefixesPartialOverlapSmallNetworks(t *testing.T) {
	g := NewWithT(t)

	pool, err := NewPrefixPool("10.20.0.0/16")
	g.Expect(err).To(BeNil())
	excludedPrefix := []string{"10.20.1.0/30", "10.20.10.0/30", "10.20.20.0/30", "10.20.20.20/30", "10.20.40.20/30"}

	_, err = pool.ExcludePrefixes(excludedPrefix)

	g.Expect(err).To(BeNil())
	g.Expect(pool.GetPrefixes()).To(Equal([]string{"10.20.32.0/21", "10.20.40.0/28", "10.20.40.16/30", "10.20.48.0/20", "10.20.44.0/22", "10.20.42.0/23", "10.20.41.0/24", "10.20.40.128/25", "10.20.40.64/26", "10.20.40.32/27", "10.20.40.24/29", "10.20.20.16/30", "10.20.20.24/29", "10.20.16.0/22", "10.20.24.0/21", "10.20.22.0/23", "10.20.21.0/24", "10.20.20.128/25", "10.20.20.64/26", "10.20.20.32/27", "10.20.20.8/29", "10.20.20.4/30", "10.20.8.0/23", "10.20.12.0/22", "10.20.11.0/24", "10.20.10.128/25", "10.20.10.64/26", "10.20.10.32/27", "10.20.10.16/28", "10.20.10.8/29", "10.20.10.4/30", "10.20.0.0/24", "10.20.128.0/17", "10.20.64.0/18", "10.20.4.0/22", "10.20.2.0/23", "10.20.1.128/25", "10.20.1.64/26", "10.20.1.32/27", "10.20.1.16/28", "10.20.1.8/29", "10.20.1.4/30"}))
}

func TestExcludePrefixesNoOverlap(t *testing.T) {
	g := NewWithT(t)

	pool, err := NewPrefixPool("10.20.0.0/16")
	g.Expect(err).To(BeNil())
	excludedPrefix := []string{"10.32.1.0/16"}

	_, err = pool.ExcludePrefixes(excludedPrefix)

	g.Expect(pool.GetPrefixes()).To(Equal([]string{"10.20.0.0/16"}))
}

func TestExcludePrefixesFullOverlap(t *testing.T) {
	g := NewWithT(t)

	pool, err := NewPrefixPool("10.20.0.0/24")
	g.Expect(err).To(BeNil())
	excludedPrefix := []string{"10.20.1.0/16"}

	_, err = pool.ExcludePrefixes(excludedPrefix)

	g.Expect(err).To(Equal(fmt.Errorf("IPAM: The available address pool is empty, probably intersected by excludedPrefix")))
}
