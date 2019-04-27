package tests

import (
	v1 "github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis/networkservice/v1"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/registryserver/resource_cache"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
	"time"
)

func TestNsCacheConcurrentModification(t *testing.T) {
	RegisterTestingT(t)

	c := resource_cache.NewNetworkServiceCache()
	fakeRegistry := fakeRegistry{}

	stopFunc, err := c.Start(&fakeRegistry)

	Expect(stopFunc).ToNot(BeNil())
	Expect(err).To(BeNil())

	c.Add(&v1.NetworkService{ObjectMeta: metav1.ObjectMeta{Name: "ns1"}})
	stopRead := RepeatAsync(func() {
		ns := c.Get("ns1")
		Expect(ns).ShouldNot(BeNil())
	})
	defer stopRead()

	stopWrite := RepeatAsync(func() {
		c.Add(&v1.NetworkService{ObjectMeta: metav1.ObjectMeta{Name: "ns2"}})
		c.Delete("ns2")
	})
	defer stopWrite()

	time.Sleep(time.Second * 5)
}
