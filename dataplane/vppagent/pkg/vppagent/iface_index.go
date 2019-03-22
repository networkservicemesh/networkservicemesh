package vppagent

import (
	"github.com/networkservicemesh/networkservicemesh/dataplane/vppagent/pkg/converter"
	"sync"
)

type ifaceIndex struct {
	mutex sync.RWMutex
	names map[string]string
}

func newIfaceIndex() *ifaceIndex {
	return &ifaceIndex{
		names: make(map[string]string),
	}
}

func (index *ifaceIndex) loadOrStore(id, ifaceName string) (string, bool) {
	index.mutex.Lock()
	defer index.mutex.Unlock()

	if prevValue, exists := index.names[id]; exists {
		return prevValue, false
	}

	index.names[id] = ifaceName
	return ifaceName, true
}

func (index *ifaceIndex) load(id string) (string, bool) {
	index.mutex.RLock()
	defer index.mutex.RUnlock()

	value, exists := index.names[id]
	return value, exists
}

func (index *ifaceIndex) delete(id string) {
	index.mutex.Lock()
	defer index.mutex.Unlock()

	delete(index.names, id)
}

func (index *ifaceIndex) GetIfaceName(id string) string {
	result, _ := index.loadOrStore(id, converter.TempIfName())
	return result
}
