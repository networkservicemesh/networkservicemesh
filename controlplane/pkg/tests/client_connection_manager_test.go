package tests

import (
	"context"
	"testing"
	"time"

	"github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	local_connection "github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	remote_connection "github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/services"
)

func newConnection() *model.ClientConnection {
	return &model.ClientConnection{
		ConnectionID:    "1",
		ConnectionState: model.ClientConnectionHealing,
		Xcon: crossconnect.NewCrossConnect("1", "ip",
			&local_connection.Connection{
				Id: "1",
			},
			&remote_connection.Connection{
				Id:                                   "-",
				DestinationNetworkServiceManagerName: "remote_nsm",
			}),
	}
}

func waitListenersIncrement(mdl model.Model, g *gomega.GomegaWithT) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	its := 0
	for mdl.ListenerCount() == 0 {
		select {
		case <-time.After(1 * time.Millisecond):
			break
		case <-ctx.Done():
			break
		}
		its++
		if ctx.Err() != nil {
			break
		}
	}
	logrus.Infof("Iterations: %v listeners: %v", its, mdl.ListenerCount())
	g.Expect(ctx.Err()).To(gomega.BeNil())
}

type connectionError struct {
	connection *model.ClientConnection
	err        error
}

func TestWaitClientConnection(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	mdl := model.NewModel()
	con := newConnection()
	mdl.AddClientConnection(con)
	clientConnectionManager := services.NewClientConnectionManager(mdl, nil, nil)

	result := make(chan connectionError, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	go func() {
		conn, err := clientConnectionManager.WaitPendingConnections(ctx, "2", "remote_nsm")
		result <- connectionError{connection: conn, err: err}
	}()

	waitListenersIncrement(mdl, g)

	mdl.ApplyClientConnectionChanges("1", func(con *model.ClientConnection) {
		con.ConnectionState = model.ClientConnectionReady
		con.Xcon.GetRemoteDestination().Id = "2"
	})

	dst := <-result
	dstCon := dst.connection
	g.Expect(dst.err).To(gomega.BeNil())
	g.Expect(dstCon).NotTo(gomega.BeNil())
	g.Expect(dstCon.GetID()).To(gomega.Equal("1"))
}

func TestWaitClientConnectionDelete(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	mdl := model.NewModel()
	con := newConnection()
	mdl.AddClientConnection(con)
	clientConnectionManager := services.NewClientConnectionManager(mdl, nil, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	result := make(chan connectionError, 1)
	go func() {
		conn, err := clientConnectionManager.WaitPendingConnections(ctx, "2", "remote_nsm")
		result <- connectionError{connection: conn, err: err}
	}()

	waitListenersIncrement(mdl, g)

	mdl.DeleteClientConnection("1")

	dst := <-result
	dstCon := dst.connection
	g.Expect(dst.err).To(gomega.BeNil())
	g.Expect(dstCon).To(gomega.BeNil())
}
