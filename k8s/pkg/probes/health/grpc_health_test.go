package health

import (
	"net"
	"os"
	"testing"
	"time"

	"github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

func TestGrpcHealth(t *testing.T) {
	assert := gomega.NewWithT(t)
	_ = os.Remove("soc")
	sock, err := net.Listen("unix", "soc")
	assert.Expect(err).Should(gomega.BeNil())
	server := tools.NewServer()
	health := NewGrpcHealth(server, tools.NewAddr("unix", "soc"), time.Second)
	go func() {
		err := server.Serve(sock)
		if err != nil {
			t.Fatal(err.Error())
		}
	}()
	err = health.Check()
	assert.Expect(err).Should(gomega.BeNil())
	_ = os.Remove("soc")
}
