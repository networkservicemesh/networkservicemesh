package tests

import (
	v1 "github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis/networkservice/v1"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/registryserver/resource_cache"
	. "github.com/onsi/gomega"
	"testing"
	"time"
)

func TestNsmCacheGetNil(t *testing.T) {
	RegisterTestingT(t)
	c := resource_cache.NewNetworkServiceManagerCache()

	stopFunc, err := c.Start(&fakeRegistry{})
	defer stopFunc()
	Expect(stopFunc).ToNot(BeNil())
	Expect(err).To(BeNil())
	var expect *v1.NetworkServiceManager = nil
	Expect(c.Get("Justice")).Should(Equal(expect))
}

func TestNsmCacheConcurrentModification(t *testing.T) {
	RegisterTestingT(t)
	c := resource_cache.NewNetworkServiceManagerCache()

	stopFunc, err := c.Start(&fakeRegistry{})
	defer stopFunc()
	Expect(stopFunc).ToNot(BeNil())
	Expect(err).To(BeNil())

	c.Add(FakeNsm("nsm-1"))
	c.Add(FakeNsm("nsm-2"))

	stopRead := RepeatAsync(func() {
		nsm1 := c.Get("nsm-1")
		Expect(nsm1).ShouldNot(BeNil())
		Expect(nsm1.Name).Should(Equal("nsm-1"))

		c.Get("nsm-2")

	})
	defer stopRead()
	stopUpdate := RepeatAsync(func() {
		c.Update(FakeNsm("nsm-1"))
	})
	defer stopUpdate()
	stopWrite := RepeatAsync(func() {
		c.Delete("nsm-2")
		c.Add(FakeNsm("nsm-2"))

	})
	defer stopWrite()

	time.Sleep(time.Second * 5)
}
