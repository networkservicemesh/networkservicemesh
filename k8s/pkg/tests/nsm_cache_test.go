// +build unit_test

package tests

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis/networkservice/v1alpha1"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/registryserver/resourcecache"
)

func TestNsmCacheGetNil(t *testing.T) {
	g := NewWithT(t)
	c := resourcecache.NewNetworkServiceManagerCache(resourcecache.NoFilterPolicy())

	stopFunc, err := c.Start(&fakeRegistry{})
	defer stopFunc()
	g.Expect(stopFunc).ToNot(BeNil())
	g.Expect(err).To(BeNil())
	var expect *v1.NetworkServiceManager = nil
	g.Expect(c.Get("Justice")).Should(Equal(expect))
}

func TestNsmCacheConcurrentModification(t *testing.T) {
	g := NewWithT(t)
	c := resourcecache.NewNetworkServiceManagerCache(resourcecache.NoFilterPolicy())

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

	<-time.After(time.Second)
}

func TestNSMCacheAddResourceWithNamespace(t *testing.T) {
	g := NewWithT(t)
	nsmCache := resourcecache.NewNetworkServiceManagerCache(resourcecache.FilterByNamespacePolicy("1", func(resource interface{}) string {
		return resource.(*v1.NetworkServiceManager).Namespace
	}))
	reg := fakeRegistry{}

	stopFunc, err := nsmCache.Start(&reg)
	g.Expect(stopFunc).ToNot(BeNil())
	g.Expect(err).To(BeNil())
	defer stopFunc()
	reg.Add(&v1.NetworkServiceManager{ObjectMeta: metav1.ObjectMeta{Name: "nsm1"}})
	g.Expect(nsmCache.Get("nsm1")).Should(BeNil())
	reg.Add(&v1.NetworkServiceManager{ObjectMeta: metav1.ObjectMeta{Name: "nsm1", Namespace: "1"}})
	g.Expect(nsmCache.Get("nsm1")).ShouldNot(BeNil())
}

func TestNsmCacheStartWithInit(t *testing.T) {
	g := NewWithT(t)
	c := resourcecache.NewNetworkServiceManagerCache(resourcecache.NoFilterPolicy())

	init := []v1.NetworkServiceManager{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "nsm-1"},
			Spec:       v1.NetworkServiceManagerSpec{URL: "1.1.1.1"},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "nsm-2"},
			Spec:       v1.NetworkServiceManagerSpec{URL: "2.2.2.2"},
		},
	}
	stopFunc, err := c.Start(&fakeRegistry{}, init...)
	defer stopFunc()
	g.Expect(stopFunc).ToNot(BeNil())
	g.Expect(err).To(BeNil())

	g.Expect(c.Get("nsm-1").Name).To(Equal("nsm-1"))
	g.Expect(c.Get("nsm-2").Name).To(Equal("nsm-2"))
}
