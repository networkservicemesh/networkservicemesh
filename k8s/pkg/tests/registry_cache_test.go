package tests

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"time"

	. "github.com/onsi/gomega"

	v1 "github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis/networkservice/v1alpha1"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice/clientset/versioned"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice/clientset/versioned/scheme"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/registryserver"

	"net/http"
	"sync"
	"testing"
)

func readNsm(reader io.Reader) (v1.NetworkServiceManager, error) {
	nsm := v1.NetworkServiceManager{}
	msg, err := ioutil.ReadAll(reader)
	if err != nil {
		return nsm, err
	}

	err = json.Unmarshal(msg, &nsm)
	return nsm, err
}

func fakeNsmRest(g *WithT, serverData *sync.Map) *FakeRest {
	result := NewFakeRest(v1.SchemeGroupVersion, scheme.Codecs)
	result.MockGet("/networkserviceendpoints", func(r *http.Request, resource string) (response *http.Response, e error) {
		return Ok([]v1.NetworkServiceEndpoint{}), nil
	})
	result.MockGet("/namespaces/default/networkserviceendpoints", func(r *http.Request, resource string) (response *http.Response, e error) {
		return Ok(v1.NetworkServiceEndpointList{}), nil
	})
	result.MockGet("/namespaces/default/networkservicemanagers", func(r *http.Request, resource string) (response *http.Response, e error) {
		if val, ok := serverData.Load(resource); ok {
			return Ok(val), nil
		}
		return Ok(v1.NetworkServiceManagerList{}), nil
	})
	result.MockGet("/networkservices", func(r *http.Request, resource string) (response *http.Response, e error) {
		return Ok([]v1.NetworkService{}), nil
	})
	result.MockGet("/namespaces/default/networkservices", func(r *http.Request, resource string) (response *http.Response, e error) {
		return Ok(v1.NetworkServiceList{}), nil
	})
	result.MockGet("/networkservicemanagers", func(r *http.Request, resource string) (response *http.Response, e error) {
		return Ok([]v1.NetworkService{}), nil
	})
	result.MockPost("/namespaces/default/networkservicemanagers", func(r *http.Request, resource string) (response *http.Response, e error) {
		nsm, err := readNsm(r.Body)
		g.Expect(err).To(BeNil())
		if val, ok := serverData.Load(nsm.Name); ok {
			return AlreadyExist(val), nil
		}
		nsm.ResourceVersion = "1"
		serverData.Store(nsm.Name, nsm)
		return Ok(nsm), nil
	})
	result.MockPut("/namespaces/default/networkservicemanagers", func(r *http.Request, resource string) (response *http.Response, e error) {
		nsm, err := readNsm(r.Body)
		g.Expect(err).To(BeNil())
		if nsm.ResourceVersion == "0" {
			return BadVersion(nsm), nil
		}
		if _, ok := serverData.Load(resource); ok {
			serverData.Store(nsm.Name, nsm)
			return Ok(nsm), nil
		}
		return NotFound(nsm), nil
	})
	return result
}

func TestCreateOrUpdateNetworkServiceManager(t *testing.T) {
	g := NewWithT(t)
	nsm := FakeNsm("fake")
	nsm.ResourceVersion = "1"
	serverData := sync.Map{}
	serverData.Store("fake", nsm)
	fakeRest := fakeNsmRest(g, &serverData)
	cache := registryserver.NewRegistryCache(versioned.New(fakeRest), nil)
	err := cache.Start()
	g.Expect(err).Should(BeNil())
	_, err = cache.CreateOrUpdateNetworkServiceManager(FakeNsm("fake"))
	defer cache.Stop()
	g.Expect(err).Should(BeNil())

}
func TestConcurrentCreateOrUpdateNetworkServiceManager(t *testing.T) {
	g := NewWithT(t)
	serverData := sync.Map{}
	fakeRest := fakeNsmRest(g, &serverData)
	for i := 0; i < 10; i++ {
		cache := registryserver.NewRegistryCache(versioned.New(fakeRest), nil)
		err := cache.Start()
		g.Expect(err).Should(BeNil())
		defer cache.Stop()
		stopClient1 := RepeatAsync(func() {
			nsm := FakeNsm("fake")
			_, err := cache.CreateOrUpdateNetworkServiceManager(nsm)
			g.Expect(err).Should(BeNil())
		})
		defer stopClient1()
		stopClient2 := RepeatAsync(func() {
			nsm := FakeNsm("fake")
			_, err := cache.CreateOrUpdateNetworkServiceManager(nsm)
			g.Expect(err).Should(BeNil())
		})
		defer stopClient2()
		time.Sleep(time.Microsecond * 500)
	}
}

func TestUpdatingExistingNetworkServiceManager(t *testing.T) {
	g := NewWithT(t)
	serverData := sync.Map{}
	fakeRest := fakeNsmRest(g, &serverData)
	cache := registryserver.NewRegistryCache(versioned.New(fakeRest), nil)
	err := cache.Start()
	g.Expect(err).Should(BeNil())
	defer cache.Stop()
	nsm := FakeNsm("fake")
	_, err = cache.CreateOrUpdateNetworkServiceManager(nsm)
	g.Expect(err).Should(BeNil())
	nsm.Status.URL = "update"
	_, err = cache.CreateOrUpdateNetworkServiceManager(nsm)
	g.Expect(err).Should(BeNil())
	val, ok := serverData.Load("fake")
	g.Expect(ok).Should(Equal(true))
	g.Expect(val.(v1.NetworkServiceManager).Status.URL).Should(Equal("update"))

}
func TestUpdatingNotExistingNetworkServiceManager(t *testing.T) {
	g := NewWithT(t)
	serverData := sync.Map{}
	fakeRest := fakeNsmRest(g, &serverData)
	cache := registryserver.NewRegistryCache(versioned.New(fakeRest), nil)
	err := cache.Start()
	g.Expect(err).Should(BeNil())
	defer cache.Stop()
	nsm := FakeNsm("fake")
	nsm.ResourceVersion = "1"
	serverData.Store(nsm.Name, nsm.DeepCopy())
	g.Expect(err).Should(BeNil())
	nsm.Status.URL = "update"
	_, err = cache.CreateOrUpdateNetworkServiceManager(nsm)
	g.Expect(err).Should(BeNil())
	val, ok := serverData.Load("fake")
	g.Expect(ok).Should(Equal(true))
	g.Expect(val.(v1.NetworkServiceManager).Status.URL).Should(Equal("update"))
}
