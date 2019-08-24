package tests

import (
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/registryserver/resourcecache"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis/networkservice/v1alpha1"
)

func TestNsCacheConcurrentModification(t *testing.T) {
	g := NewWithT(t)

	c := resourcecache.NewNetworkServiceCache(resourcecache.NoFilterPolicy())
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

	<-time.After(time.Second)
}
