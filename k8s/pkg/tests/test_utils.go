package tests

import (
	"bytes"
	"encoding/json"
	v1 "github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis/networkservice/v1"
	. "github.com/onsi/gomega"
	"io"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
)

func Body(content interface{}) io.ReadCloser {
	msg, err := json.Marshal(content)
	Expect(err).Should(BeNil())
	return ioutil.NopCloser(bytes.NewReader(msg))
}
func Ok(content interface{}) *http.Response {
	return Status(http.StatusOK, content)
}

func BadVersion(content interface{}) *http.Response {
	return &http.Response{
		StatusCode: http.StatusConflict,
		Status:     "Bad version",
		Body:       Body(content),
	}
}
func NotFound(content interface{}) *http.Response {
	return &http.Response{
		StatusCode: http.StatusNotFound,
		Status:     "Not found",
		Body:       Body(content),
	}
}
func AlreadyExist(content interface{}) *http.Response {
	return &http.Response{
		StatusCode: http.StatusConflict,
		Status:     "AlreadyExists",
		Body:       Body(content),
	}
}

func Status(status int, content interface{}) *http.Response {
	msg, err := json.Marshal(content)
	Expect(err).Should(BeNil())
	return &http.Response{
		StatusCode: status,
		Body:       ioutil.NopCloser(bytes.NewReader(msg)),
	}
}
func RepeatAsync(operation func()) (stopFunc func()) {
	Expect(operation).ShouldNot(BeNil())
	stopCh := make(chan struct{})
	stopFunc = func() {
		close(stopCh)
	}
	go func() {
		for {
			select {
			case <-stopCh:
				return
			default:
				operation()
			}
		}
	}()
	return
}
func FakeNsm(name string) *v1.NetworkServiceManager {
	return &v1.NetworkServiceManager{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			ResourceVersion: "0",
		},
	}
}
