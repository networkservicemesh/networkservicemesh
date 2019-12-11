// +build unit_test

package health

import (
	"net/http"
	"strings"
	"testing"

	"github.com/pkg/errors"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools"

	"github.com/onsi/gomega"
)

type mockClient struct {
}

func (c *mockClient) Do(r *http.Request) (*http.Response, error) {
	if strings.HasSuffix(r.URL.Path, "/health") {
		return &http.Response{}, nil
	}
	return nil, errors.New("wrong path")
}

func TestServeMuxHealth(t *testing.T) {
	assert := gomega.NewWithT(t)
	mux := http.NewServeMux()
	health := newHTTPServeMuxHealth(tools.NewAddr("http", ":443"), mux, &mockClient{})
	err := health.Check()
	assert.Expect(err).Should(gomega.BeNil())
}
