package tests

import (
	"github.com/networkservicemesh/networkservicemesh/dataplane/pkg/memifproxy"
	. "github.com/onsi/gomega"
	"testing"
)

func TestDataplaneCrossConnectBasic(t *testing.T) {
	RegisterTestingT(t)
	proxy := memifproxy.NewCustomProxy("source.sock", "target.sock", "unix")
	for i := 0; i < 10; i++ {
		err := proxy.Start()
		Expect(err).To(BeNil())
		err = proxy.Stop();
		Expect(err).To(BeNil())
	}
}
