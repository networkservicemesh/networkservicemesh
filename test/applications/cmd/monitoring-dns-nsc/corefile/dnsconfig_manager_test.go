package corefile

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/onsi/gomega"
	"io/ioutil"
	"os"
	"testing"
)

func TestDNSConfigManager(t *testing.T) {
	const coreFilePath = "Corefile"
	gomega.RegisterTestingT(t)
	mngr, err := NewDNSConfigManager(coreFilePath)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(mngr).ShouldNot(gomega.BeNil())
	dnsConfig := connectioncontext.DNSContext{
		DnsServerIps: []string{"9.9.9.9"},
	}
	gomega.Expect(mngr.UpdateDNSConfig(&dnsConfig)).Should(gomega.BeNil())

	buf, err := ioutil.ReadFile(coreFilePath)
	gomega.Expect(err).Should(gomega.BeNil())

	expected := `.:54 {
	log
	reload 5s
}
. {
	log
	forward . 9.9.9.9
}
`
	gomega.Expect(string(buf)).Should(gomega.Equal(expected))
	gomega.Expect(mngr.RemoveDNSConfig(&dnsConfig)).Should(gomega.BeNil())
	buf, err = ioutil.ReadFile(coreFilePath)
	expected = `.:54 {
	log
	reload 5s
}
`
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(string(buf)).Should(gomega.Equal(expected))
	gomega.Expect(os.Remove(coreFilePath)).Should(gomega.BeNil())
}
