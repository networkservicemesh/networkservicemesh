package tests

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis/networkservice/v1alpha1"

	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/registryserver/resourcecache"
)

func TestK8sRegistryAdd(t *testing.T) {
	g := NewWithT(t)

	fakeRegistry := fakeRegistry{}
	nseCache := resourcecache.NewNetworkServiceEndpointCache(resourcecache.NoFilterPolicy())

	stopFunc, err := nseCache.Start(&fakeRegistry)

	g.Expect(stopFunc).ToNot(BeNil())
	g.Expect(err).To(BeNil())

	nse := newTestNse("nse1", "ns1")
	fakeRegistry.Add(nse)

	endpointList := getEndpoints(nseCache, "ns1", 1)
	g.Expect(len(endpointList)).To(Equal(1))
	g.Expect(endpointList[0].Name).To(Equal("nse1"))
}

func TestNseCacheConcurrentModification(t *testing.T) {
	g := NewWithT(t)
	fakeRegistry := fakeRegistry{}
	c := resourcecache.NewNetworkServiceEndpointCache(resourcecache.NoFilterPolicy())

	stopFunc, err := c.Start(&fakeRegistry)
	defer stopFunc()
	g.Expect(stopFunc).ToNot(BeNil())
	g.Expect(err).To(BeNil())

	c.Add(newTestNse("nse1", "ns1"))
	c.Add(newTestNse("nse2", "ns2"))

	stopRead := RepeatAsync(func() {
		c.Get("nse1")
		c.Get("nse2")
		c.GetByNetworkService("ms1")
	})
	defer stopRead()
	stopWrite := RepeatAsync(func() {
		c.Delete("nsm2")
		c.Add(newTestNse("nse2", "ns2"))
	})
	defer stopWrite()
	<-time.After(time.Second)
}

func TestNsmdRegistryAdd(t *testing.T) {
	g := NewWithT(t)

	fakeRegistry := fakeRegistry{}
	nseCache := resourcecache.NewNetworkServiceEndpointCache(resourcecache.NoFilterPolicy())

	stopFunc, err := nseCache.Start(&fakeRegistry)

	g.Expect(stopFunc).ToNot(BeNil())
	g.Expect(err).To(BeNil())

	nse := newTestNse("nse1", "ns1")
	nseCache.Add(nse)

	endpointList := getEndpoints(nseCache, "ns1", 1)
	g.Expect(len(endpointList)).To(Equal(1))
	g.Expect(endpointList[0].Name).To(Equal("nse1"))
}

func TestRegistryDelete(t *testing.T) {
	g := NewWithT(t)

	fakeRegistry := fakeRegistry{}
	nseCache := resourcecache.NewNetworkServiceEndpointCache(resourcecache.NoFilterPolicy())

	stopFunc, err := nseCache.Start(&fakeRegistry)

	g.Expect(stopFunc).ToNot(BeNil())
	g.Expect(err).To(BeNil())

	nse1 := newTestNse("nse1", "ns1")
	nse2 := newTestNse("nse2", "ns1")
	nse3 := newTestNse("nse3", "ns2")

	fakeRegistry.Add(nse1)
	fakeRegistry.Add(nse2)
	fakeRegistry.Add(nse3)

	endpointList1 := getEndpoints(nseCache, "ns1", 2)
	g.Expect(len(endpointList1)).To(Equal(2))
	endpointList2 := getEndpoints(nseCache, "ns2", 1)
	g.Expect(len(endpointList2)).To(Equal(1))

	fakeRegistry.Delete(nse3)
	endpointList3 := getEndpoints(nseCache, "ns2", 0)
	g.Expect(len(endpointList3)).To(Equal(0))
}

func getEndpoints(nseCache *resourcecache.NetworkServiceEndpointCache,
	networkServiceName string, expectedLength int) []*v1.NetworkServiceEndpoint {
	var endpointList []*v1.NetworkServiceEndpoint
	for attempt := 0; attempt < 10; <-time.After(300 * time.Millisecond) {
		attempt++
		endpointList = nseCache.GetByNetworkService(networkServiceName)
		if len(endpointList) == expectedLength {
			logrus.Infof("Attempt: %v", attempt)
			break
		}
	}
	return endpointList
}

func newTestNse(name string, networkServiceName string) *v1.NetworkServiceEndpoint {
	return &v1.NetworkServiceEndpoint{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1.NetworkServiceEndpointSpec{
			NetworkServiceName: networkServiceName,
			Payload:            "IP",
			NsmName:            "nsm1",
		},
		Status: v1.NetworkServiceEndpointStatus{
			State: v1.RUNNING,
		},
	}
}
