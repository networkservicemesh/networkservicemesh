package health

import (
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools"

	"github.com/onsi/gomega"
)

func TestServeMuxHealth(t *testing.T) {
	assert := gomega.NewWithT(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/product", testFuncHandler(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("ok"))
	}))
	listener, err := net.Listen("tcp", ":0")
	assert.Expect(err).Should(gomega.BeNil())
	port := listener.Addr().(*net.TCPAddr).Port
	addr := fmt.Sprintf(":%v", port)
	err = listener.Close()
	assert.Expect(err).Should(gomega.BeNil())

	go func() {
		err := http.ListenAndServe(addr, mux)
		if err != nil {
			t.Fatal(err.Error())
		}
	}()

	health := NewHttpServeMuxHealth(tools.NewAddr("http", addr), mux, time.Second)
	err = health.Check()
	assert.Expect(err).Should(gomega.BeNil())
}

type testFuncHandler func(http.ResponseWriter, *http.Request)

func (t testFuncHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t(w, r)
}
