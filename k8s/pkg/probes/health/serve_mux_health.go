package health

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/sirupsen/logrus"
)

type httpClient interface {
	Do(r *http.Request) (*http.Response, error)
}

//NewHTTPServeMuxHealth creates health checker for http based applications
func NewHTTPServeMuxHealth(addr net.Addr, mux *http.ServeMux, timeout time.Duration) ApplicationHealth {
	client := http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	return newHTTPServeMuxHealth(addr, mux, &client)
}

func newHTTPServeMuxHealth(addr net.Addr, mux *http.ServeMux, client httpClient) ApplicationHealth {
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte("ok"))
		if err != nil {
			logrus.Errorf("HttpHealthChecker: write error%v", err)
		}
	})
	u, err := url.Parse(fmt.Sprintf("%v://%v/health", addr.Network(), addr.String()))
	return NewApplicationHealthFunc(
		func() error {
			if err != nil {
				return err
			}
			req := &http.Request{
				Method: "GET",
				URL:    u,
			}
			_, err = client.Do(req)
			return err
		})
}
