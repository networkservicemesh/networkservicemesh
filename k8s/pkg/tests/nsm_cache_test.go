package tests

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis/networkservice/v1alpha1"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/registryserver/resource_cache"
)

func TestNsmCacheGetNil(t *testing.T) {
	g := NewWithT(t)
	c := resource_cache.NewNetworkServiceManagerCache()

	stopFunc, err := c.Start(&fakeRegistry{})
	defer stopFunc()
	g.Expect(stopFunc).ToNot(BeNil())
	g.Expect(err).To(BeNil())
	var expect *v1.NetworkServiceManager = nil
	g.Expect(c.Get("Justice")).Should(Equal(expect))
}

func TestNsmCacheConcurrentModification(t *testing.T) {
	g := NewWithT(t)
	c := resource_cache.NewNetworkServiceManagerCache()

	stopFunc, err := c.Start(&fakeRegistry{})
	defer stopFunc()
	g.Expect(stopFunc).ToNot(BeNil())
	g.Expect(err).To(BeNil())

	c.Add(FakeNsm("nsm-1"))
	c.Add(FakeNsm("nsm-2"))

	stopRead := RepeatAsync(func() {
		nsm1 := c.Get("nsm-1")
		g.Expect(nsm1).ShouldNot(BeNil())
		g.Expect(nsm1.Name).Should(Equal("nsm-1"))

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

func TestNsmCacheStartWithInit(t *testing.T) {
	g := NewWithT(t)
	c := resource_cache.NewNetworkServiceManagerCache()

	init := []v1.NetworkServiceManager{
		{
			ObjectMeta: v12.ObjectMeta{Name: "nsm-1"},
			Status:     v1.NetworkServiceManagerStatus{URL: "1.1.1.1"},
		},
		{
			ObjectMeta: v12.ObjectMeta{Name: "nsm-2"},
			Status:     v1.NetworkServiceManagerStatus{URL: "2.2.2.2"},
		},
	}
	stopFunc, err := c.Start(&fakeRegistry{}, init...)
	defer stopFunc()
	g.Expect(stopFunc).ToNot(BeNil())
	g.Expect(err).To(BeNil())

	g.Expect(c.Get("nsm-1").Name).To(Equal("nsm-1"))
	g.Expect(c.Get("nsm-2").Name).To(Equal("nsm-2"))
}
