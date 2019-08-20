package tests

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis/networkservice/v1alpha1"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/registryserver/resource_cache"
)

func TestNsCacheConcurrentModification(t *testing.T) {
	g := NewWithT(t)

	c := resource_cache.NewNetworkServiceCache("")
	fakeRegistry := fakeRegistry{}

	stopFunc, err := c.Start(&fakeRegistry)

	g.Expect(stopFunc).ToNot(BeNil())
	g.Expect(err).To(BeNil())

	c.Add(&v1.NetworkService{ObjectMeta: metav1.ObjectMeta{Name: "ns1"}})
	stopRead := RepeatAsync(func() {
		ns := c.Get("ns1")
		g.Expect(ns).ShouldNot(BeNil())
	})
	defer stopRead()

	stopWrite := RepeatAsync(func() {
		c.Add(&v1.NetworkService{ObjectMeta: metav1.ObjectMeta{Name: "ns2"}})
		c.Delete("ns2")
	})
	defer stopWrite()

	time.Sleep(time.Second * 5)
}

func TestNSCacheAddResourceWithNamespace(t *testing.T) {
	g := NewWithT(t)
	nsCache := resource_cache.NewNetworkServiceCache("1")
	reg := fakeRegistry{}

	stopFunc, err := nsCache.Start(&reg)
	g.Expect(stopFunc).ToNot(BeNil())
	g.Expect(err).To(BeNil())
	defer stopFunc()
	reg.Add(&v1.NetworkService{ObjectMeta: metav1.ObjectMeta{Name: "ns1"}, Spec: v1.NetworkServiceSpec{}})
	g.Expect(nsCache.Get("ns1")).Should(BeNil())
	reg.Add(&v1.NetworkService{ObjectMeta: metav1.ObjectMeta{Name: "ns1", Namespace: "1"}, Spec: v1.NetworkServiceSpec{}})
	g.Expect(nsCache.Get("ns1")).ShouldNot(BeNil())
}
