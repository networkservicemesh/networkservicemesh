package tests

import (
	"context"
	"testing"
	"time"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	remote_networkservice "github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/networkservice"

	"github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	unified "github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/services"
)

func newConnection() *model.ClientConnection {
	return &model.ClientConnection{
		ConnectionID:    "1",
		ConnectionState: model.ClientConnectionHealing,
		Xcon: crossconnect.NewCrossConnect("1", "ip",
			&unified.Connection{
				Id: "1",
				NetworkServiceManagers: []string{
					"local_nsm",
				},
			},
			&unified.Connection{
				Id: "-",
				NetworkServiceManagers: []string{
					"local_nsm", "remote_nsm",
				},
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
		case <-ctx.Done():
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
	mdl.AddClientConnection(context.Background(), con)
	clientConnectionManager := services.NewClientConnectionManager(mdl, nil, nil)

	result := make(chan connectionError, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	go func() {
		conn, err := clientConnectionManager.WaitPendingConnections(ctx, "2", "remote_nsm")
		result <- connectionError{connection: conn, err: err}
	}()

	waitListenersIncrement(mdl, g)

	mdl.ApplyClientConnectionChanges(context.Background(), "1", func(con *model.ClientConnection) {
		con.ConnectionState = model.ClientConnectionReady
		con.Xcon.Destination.Id = "2"
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
	mdl.AddClientConnection(context.Background(), con)
	clientConnectionManager := services.NewClientConnectionManager(mdl, nil, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	result := make(chan connectionError, 1)
	go func() {
		conn, err := clientConnectionManager.WaitPendingConnections(ctx, "2", "remote_nsm")
		result <- connectionError{connection: conn, err: err}
	}()

	waitListenersIncrement(mdl, g)

	mdl.DeleteClientConnection(context.Background(), "1")

	dst := <-result
	dstCon := dst.connection
	g.Expect(dst.err).To(gomega.BeNil())
	g.Expect(dstCon).To(gomega.BeNil())
}

func TestGetClientConnectionBySource(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	mdl := model.NewModel()
	createConnections(mdl)
	clientConnectionManager := services.NewClientConnectionManager(mdl, nil, nil)

	c1 := clientConnectionManager.GetClientConnectionBySource("nsm1")
	g.Expect(len(c1)).To(gomega.Equal(1))
	g.Expect(c1[0].ConnectionID).To(gomega.Equal("3"))
}

func TestGetClientConnectionByRemoteDst(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	mdl := model.NewModel()
	createConnections(mdl)
	clientConnectionManager := services.NewClientConnectionManager(mdl, nil, nil)

	c1 := clientConnectionManager.GetClientConnectionByRemoteDst("3", "remote_nsm")
	g.Expect(c1).ToNot(gomega.BeNil())
	g.Expect(c1.ConnectionID).To(gomega.Equal("2"))
}

func TestGetClientConnectionByLocalDst(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	mdl := model.NewModel()
	createConnections(mdl)
	clientConnectionManager := services.NewClientConnectionManager(mdl, nil, nil)

	c1 := clientConnectionManager.GetClientConnectionByLocalDst("4")
	g.Expect(c1).ToNot(gomega.BeNil())
	g.Expect(c1.ConnectionID).To(gomega.Equal("3"))
}

func TestGetClientConnectionByRemote(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	mdl := model.NewModel()
	createConnections(mdl)
	clientConnectionManager := services.NewClientConnectionManager(mdl, nil, nil)

	c1 := clientConnectionManager.GetClientConnectionByRemote(&registry.NetworkServiceManager{
		Name: "rnsm",
	})
	g.Expect(len(c1)).To(gomega.Equal(1))
	g.Expect(c1[0].ConnectionID).To(gomega.Equal("6"))
}

func TestGetClientConnectionByXCon(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	mdl := model.NewModel()
	createConnections(mdl)
	clientConnectionManager := services.NewClientConnectionManager(mdl, nil, nil)
	cc := &model.ClientConnection{
		ConnectionID:    "5",
		ConnectionState: model.ClientConnectionHealing,
		Xcon: crossconnect.NewCrossConnect("5", "IP",
			&unified.Connection{
				Id:             "2",
				NetworkService: "s2",
				NetworkServiceManagers: []string{
					"local_nsm",
				},
			},
			&unified.Connection{
				Id: "3",
				NetworkServiceManagers: []string{
					"local_nsm", "remote_nsm",
				},
			}),
	}
	clientConnectionManager.MarkConnectionAdded(cc)
	clientConnectionManager.MarkConnectionDeleted(cc)

	c1 := clientConnectionManager.GetClientConnectionByXcon(cc.Xcon)
	g.Expect(c1).ToNot(gomega.BeNil())
	g.Expect(c1.ConnectionID).To(gomega.Equal("5"))
}

func createConnections(mdl model.Model) {
	mdl.AddClientConnection(context.Background(), &model.ClientConnection{
		ConnectionID:    "1",
		ConnectionState: model.ClientConnectionHealing,
		Xcon:            nil,
	})

	mdl.AddClientConnection(context.Background(), &model.ClientConnection{
		ConnectionID:    "2",
		ConnectionState: model.ClientConnectionHealing,
		Request: &remote_networkservice.NetworkServiceRequest{
			Connection:           nil,
			MechanismPreferences: nil,
		},
		Xcon: crossconnect.NewCrossConnect("2", "ip",
			&unified.Connection{
				Id:             "2",
				NetworkService: "s2",
				NetworkServiceManagers: []string{
					"local_nsm",
				},
			},
			&unified.Connection{
				Id: "3",
				NetworkServiceManagers: []string{
					"local_nsm", "remote_nsm",
				},
			}),
	})
	mdl.AddClientConnection(context.Background(), &model.ClientConnection{
		ConnectionID:    "3",
		ConnectionState: model.ClientConnectionHealing,
		Request: &remote_networkservice.NetworkServiceRequest{
			Connection:           nil,
			MechanismPreferences: nil,
		},
		Xcon: crossconnect.NewCrossConnect("3", "ip",
			&unified.Connection{
				Id:             "3",
				NetworkService: "s2",
				NetworkServiceManagers: []string{
					"nsm1", "remote_nsm",
				},
			},
			&unified.Connection{
				Id: "4",
				NetworkServiceManagers: []string{
					"local_nsm",
				},
			}),
	})
	mdl.AddClientConnection(context.Background(), &model.ClientConnection{
		ConnectionID:    "6",
		ConnectionState: model.ClientConnectionHealing,
		Xcon:            nil,
		RemoteNsm: &registry.NetworkServiceManager{
			Name: "rnsm",
		},
	})
}
