package health

import (
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
	s := http.Server{
		Addr:    ":5000",
		Handler: mux,
	}

	go func() {
		err := s.ListenAndServe()
		if err != nil {
			t.Fatal(err.Error())
		}
	}()

	health := NewHttpServeMuxHealth(tools.NewAddr("http", ":5000"), mux, time.Second)
	err := health.Check()
	assert.Expect(err).Should(gomega.BeNil())
}

type testFuncHandler func(http.ResponseWriter, *http.Request)

func (t testFuncHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t(w, r)
}
