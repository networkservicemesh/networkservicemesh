package prefixcollector

import (
	"fmt"
	"net"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
)

type dummyResource struct {
	name   string
	subnet string
}

func (*dummyResource) GetObjectKind() schema.ObjectKind {
	panic("implement me")
}

func (*dummyResource) DeepCopyObject() runtime.Object {
	panic("implement me")
}

func keyFuncDummy(event watch.Event) (string, error) {
	return event.Object.(*dummyResource).name, nil
}

func subnetFuncDummy(event watch.Event) (*net.IPNet, error) {
	_, ipNet, _ := net.ParseCIDR(event.Object.(*dummyResource).subnet)
	return ipNet, nil
}

type dummyWatcher struct {
	eventCh chan watch.Event
}

func NewDummyWatcher() *dummyWatcher {
	return &dummyWatcher{
		eventCh: make(chan watch.Event),
	}
}

func (d *dummyWatcher) Stop() {
	panic("implement me")
}

func (d *dummyWatcher) ResultChan() <-chan watch.Event {
	return d.eventCh
}

func (d *dummyWatcher) send(t watch.EventType, dr *dummyResource) {
	d.eventCh <- watch.Event{
		Type:   t,
		Object: dr,
	}
}

func checkSubnetWatcher(t *testing.T, subnetSequence, expectedSequence []string) {
	g := NewWithT(t)

	dw := NewDummyWatcher()
	sw, err := watchSubnet(dw, keyFuncDummy, subnetFuncDummy)
	g.Expect(err).To(BeNil())

	for i := 0; i < len(subnetSequence); i++ {
		dw.send(watch.Added, &dummyResource{fmt.Sprintf("r%d", i), subnetSequence[i]})
	}

	for i := 0; i < len(expectedSequence); i++ {
		select {
		case e := <-sw.ResultChan():
			g.Expect(e.String()).To(Equal(expectedSequence[i]))
		case <-time.After(1 * time.Second):
			if expectedSequence[i] != "-" {
				logrus.Error("Timeout waiting for next subnet")
				t.Fail()
			}
		}
	}
}

func TestSimpleSubnetCollector(t *testing.T) {
	subnetSequence := []string{
		"10.20.1.0/24",
		"10.20.2.0/24",
	}
	expectedSubnets := []string{
		"10.20.1.0/24",
		"10.20.0.0/22",
	}
	checkSubnetWatcher(t, subnetSequence, expectedSubnets)
}

func TestLastIpsAlreadyInSubnet(t *testing.T) {
	subnetSequence := []string{
		"10.96.10.10/32",
		"10.98.2.0/32",
		"10.98.2.255/32",
		"10.99.1.255/32",
	}
	expectedSubnets := []string{
		"10.96.10.10/32",
		"10.96.0.0/14",
		"-",
		"-",
	}
	checkSubnetWatcher(t, subnetSequence, expectedSubnets)
}

func TestIntermediateIpsAlreadyInSubnet(t *testing.T) {
	subnetSequence := []string{
		"10.96.10.10/32",
		"10.98.2.0/32",
		"10.98.2.255/32",
		"10.99.1.255/32",
		"10.104.1.255/32",
	}
	expectedSubnets := []string{
		"10.96.10.10/32",
		"10.96.0.0/14",
		"10.96.0.0/12",
	}
	checkSubnetWatcher(t, subnetSequence, expectedSubnets)
}

func TestIpv6(t *testing.T) {
	subnetSequence := []string{
		"100::1:0/112",
		"100::2:0/112",
	}
	expectedSubnets := []string{
		"100::1:0/112",
		"100::/110",
	}
	checkSubnetWatcher(t, subnetSequence, expectedSubnets)
}
