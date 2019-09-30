package tests

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/applications/nsmrs/pkg/serviceregistryserver"
)

func TestNSMRSCacheAdd(t *testing.T) {
	g := NewWithT(t)

	cache := serviceregistryserver.NewNSERegistryCache()

	nse := newTestNse("nse1", "ns1")

	_, err := cache.AddNetworkServiceEndpoint(&serviceregistryserver.NSECacheEntry{
		Nse:     nse,
		Monitor: nil,
	})
	g.Expect(err).To(BeNil())

	endpointList := cache.GetEndpointsByNs("ns1")
	g.Expect(len(endpointList)).To(Equal(1))
	g.Expect(endpointList[0].Nse.NetworkServiceEndpoint.Name).To(Equal("nse1"))
}

func TestNSMRSCacheDelete(t *testing.T) {
	g := NewWithT(t)

	cache := serviceregistryserver.NewNSERegistryCache()

	nse := newTestNse("nse1", "ns1")

	_, err := cache.AddNetworkServiceEndpoint(&serviceregistryserver.NSECacheEntry{
		Nse:     nse,
		Monitor: nil,
	})
	g.Expect(err).To(BeNil())
	endpointList := cache.GetEndpointsByNs("ns1")
	g.Expect(len(endpointList)).To(Equal(1))

	endpoint, err := cache.DeleteNetworkServiceEndpoint("nse1")
	g.Expect(err).To(BeNil())
	g.Expect(endpoint.Nse.NetworkServiceEndpoint.Name).To(Equal("nse1"))

	endpointList = cache.GetEndpointsByNs("ns1")
	g.Expect(len(endpointList)).To(Equal(0))
}

func TestNSMRSCacheNSECollision(t *testing.T) {
	g := NewWithT(t)

	cache := serviceregistryserver.NewNSERegistryCache()
	nse1 := newTestNse("nse1", "ns1")
	_, err := cache.AddNetworkServiceEndpoint(&serviceregistryserver.NSECacheEntry{
		Nse:     nse1,
		Monitor: nil,
	})
	g.Expect(err).To(BeNil())

	nse2 := newTestNse("nse2", "ns2")
	_, err = cache.AddNetworkServiceEndpoint(&serviceregistryserver.NSECacheEntry{
		Nse:     nse2,
		Monitor: nil,
	})
	g.Expect(err).To(BeNil())

	nse1clone := newTestNse("nse1", "ns1")
	_, err = cache.AddNetworkServiceEndpoint(&serviceregistryserver.NSECacheEntry{
		Nse:     nse1clone,
		Monitor: nil,
	})
	g.Expect(err.Error()).To(ContainSubstring("already exists"))
}

func TestNSMRSCacheNSCollision(t *testing.T) {
	g := NewWithT(t)

	cache := serviceregistryserver.NewNSERegistryCache()
	nse1 := newTestNseWithPayload("nse1", "ns", "IP")
	_, err := cache.AddNetworkServiceEndpoint(&serviceregistryserver.NSECacheEntry{
		Nse:     nse1,
		Monitor: nil,
	})
	g.Expect(err).To(BeNil())

	nse2 := newTestNseWithPayload("nse2", "ns", "IP")
	_, err = cache.AddNetworkServiceEndpoint(&serviceregistryserver.NSECacheEntry{
		Nse:     nse2,
		Monitor: nil,
	})
	g.Expect(err).To(BeNil())

	nse1clone := newTestNseWithPayload("nse3", "ns", "TCP")
	_, err = cache.AddNetworkServiceEndpoint(&serviceregistryserver.NSECacheEntry{
		Nse:     nse1clone,
		Monitor: nil,
	})
	g.Expect(err.Error()).To(ContainSubstring("network service already exists with different parameters"))
}
