// +build basic

package nsmd_integration_tests

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	utils "github.com/networkservicemesh/networkservicemesh/test/integration/dataplane_test_utils"
	. "github.com/onsi/gomega"
	"testing"
)

func TestDataplaneCrossConnectBasic(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	fixture := utils.CreateLocalFixture(defaultTimeout)
	defer fixture.Cleanup()

	conn := fixture.RequestDefaultKernelConnection()
	fixture.VerifyKernelConnection(conn)
}

func TestDataplaneCrossConnectMultiple(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	fixture := utils.CreateLocalFixture(defaultTimeout)
	defer fixture.Cleanup()

	first := fixture.RequestKernelConnection("id-1", "if1", "10.30.1.1/30", "10.30.1.2/30")
	second := fixture.RequestKernelConnection("id-2", "if2", "10.30.2.1/30", "10.30.2.2/30")
	fixture.VerifyKernelConnection(first)
	fixture.VerifyKernelConnection(second)
}

func TestDataplaneCrossConnectUpdate(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	fixture := utils.CreateLocalFixture(defaultTimeout)
	defer fixture.Cleanup()

	const someId = "some-id"

	orig := fixture.RequestKernelConnection(someId, "if1", "10.30.1.1/30", "10.30.1.2/30")
	fixture.VerifyKernelConnection(orig)

	updated := fixture.RequestKernelConnection(someId, "if2", "10.30.2.1/30", "10.30.2.2/30")
	fixture.VerifyKernelConnection(updated)
	fixture.VerifyKernelConnectionClosed(orig)
}

func TestDataplaneCrossConnectReconnect(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	fixture := utils.CreateLocalFixture(defaultTimeout)
	defer fixture.Cleanup()

	conn := fixture.RequestDefaultKernelConnection()
	fixture.VerifyKernelConnection(conn)

	fixture.CloseConnection(conn)
	fixture.VerifyKernelConnectionClosed(conn)

	conn = fixture.Dataplane.Request(conn) // request the same connection
	fixture.VerifyKernelConnection(conn)
}

func TestDataplaneCrossConnectRepeat(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	fixture := utils.CreateLocalFixture(defaultTimeout)
	defer fixture.Cleanup()

	conn := fixture.RequestDefaultKernelConnection()
	fixture.VerifyKernelConnection(conn)

	conn = fixture.Dataplane.Request(conn) // request the same connection
	fixture.VerifyKernelConnection(conn)
}

func TestDataplaneCrossConnectUpdateIp(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	fixture := utils.CreateLocalFixture(defaultTimeout)
	defer fixture.Cleanup()

	const (
		someId = "some-id"
		iface  = "iface"
	)

	orig := fixture.RequestKernelConnection(someId, iface, "10.30.1.1/30", "10.30.1.2/30")
	fixture.VerifyKernelConnection(orig)

	updated := fixture.RequestKernelConnection(someId, iface, "10.30.2.1/30", "10.30.2.2/30")
	fixture.VerifyKernelConnection(updated)
}

func TestDataplaneRemoteCrossConnectBasic(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	fixture := utils.CreateRemoteFixture(defaultTimeout)
	defer fixture.Cleanup()

	connSrc, connDst := fixture.RequestDefaultKernelConnection()
	fixture.VerifyKernelConnection(connSrc, connDst)
}

func TestDataplaneRemoteCrossConnectRepeatSource(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	fixture := utils.CreateRemoteFixture(defaultTimeout)
	defer fixture.Cleanup()

	connSrc, connDst := fixture.RequestDefaultKernelConnection()
	fixture.SourceDataplane().Request(connSrc) // request the same connection

	fixture.VerifyKernelConnection(connSrc, connDst)
}

func TestDataplaneRemoteCrossConnectRepeatDst(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	fixture := utils.CreateRemoteFixture(defaultTimeout)
	defer fixture.Cleanup()

	connSrc, connDst := fixture.RequestDefaultKernelConnection()
	fixture.DestDataplane().Request(connDst) // request the same connection

	fixture.VerifyKernelConnection(connSrc, connDst)
}

func TestDataplaneRemoteCrossConnectRecoverSrc(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	fixture := utils.CreateRemoteFixture(defaultTimeout)
	defer fixture.Cleanup()

	connSrc, connDst := fixture.RequestDefaultKernelConnection()
	fixture.SourceDataplane().KillVppAndHeal()
	fixture.HealConnectionSrc(connSrc)

	fixture.VerifyKernelConnection(connSrc, connDst)
}

func TestDataplaneRemoteCrossConnectRecoverDst(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	fixture := utils.CreateRemoteFixture(defaultTimeout)
	defer fixture.Cleanup()

	connSrc, connDst := fixture.RequestDefaultKernelConnection()
	fixture.DestDataplane().KillVppAndHeal()
	fixture.HealConnectionDst(connDst)

	fixture.VerifyKernelConnection(connSrc, connDst)
}
