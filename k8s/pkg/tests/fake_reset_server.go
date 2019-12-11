package tests

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type MethodHandler = func(r *http.Request, resource string) (response *http.Response, e error)

type FakeRest struct {
	*restClient
	handlers map[string]map[string]MethodHandler
	mutex    sync.Mutex
}

func newFakeRest(groupVersion schema.GroupVersion, serializer runtime.NegotiatedSerializer) *FakeRest {
	result := &FakeRest{
		restClient: &restClient{
			GroupVersion:         groupVersion,
			NegotiatedSerializer: serializer,
		},
		handlers: make(map[string]map[string]MethodHandler),
	}

	result.Client = createHTTPClient(func(request *http.Request) (response *http.Response, e error) {
		result.mutex.Lock()
		defer result.mutex.Unlock()
		if handlers, ok := result.handlers[request.Method]; ok {
			if handler, ok := handlers[request.URL.Path]; ok {
				return handler(request, "")
			}
			paths := strings.Split(request.URL.Path, "/")
			if len(paths) > 1 {
				subPath := strings.Join(paths[:len(paths)-1], "/")
				if handler, ok := handlers[subPath]; ok {
					return handler(request, paths[len(paths)-1])
				}
			}
		}
		panic(fmt.Sprintf("Not found handlers for %v", request))
	})
	return result
}

func (f *FakeRest) MockGet(api string, handler MethodHandler) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.getHandlersForMethod(http.MethodGet)[api] = handler
}
func (f *FakeRest) MockPost(api string, handler MethodHandler) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.getHandlersForMethod(http.MethodPost)[api] = handler
}
func (f *FakeRest) MockPut(api string, handler MethodHandler) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.getHandlersForMethod(http.MethodPut)[api] = handler
}

func (f *FakeRest) getHandlersForMethod(method string) map[string]MethodHandler {
	if m, ok := f.handlers[method]; ok {
		return m
	}
	f.handlers[method] = make(map[string]MethodHandler)
	return f.handlers[method]
}
