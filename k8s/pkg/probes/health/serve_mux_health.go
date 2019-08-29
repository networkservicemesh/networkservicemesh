package health

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"time"
)

type healthHandler struct {
}

func (h *healthHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	w.Write([]byte("ok"))
}

func NewHttpServeMuxHealth(addr string, mux *http.ServeMux, timeout time.Duration) ApplicationHealth {
	mux.Handle("/health", &healthHandler{})
	c := http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	u, err := url.Parse(addr + "/health")
	return NewApplicationHealthFunc(
		func() error {
			if err != nil {
				return err
			}
			req := &http.Request{
				Method: "GET",
				URL:    u,
			}
			_, err = c.Do(req)
			return err
		})
}
