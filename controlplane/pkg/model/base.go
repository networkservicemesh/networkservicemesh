package model

import "sync"

type ModificationHandler struct {
	AddFunc    func(new interface{})
	UpdateFunc func(old interface{}, new interface{})
	DeleteFunc func(del interface{})
}

type baseDomain struct {
	mtx      sync.RWMutex
	handlers []*ModificationHandler
}

func (b *baseDomain) resourceAdded(new interface{}) {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	for _, h := range b.handlers {
		if h.AddFunc != nil {
			h.AddFunc(new)
		}
	}
}

func (b *baseDomain) resourceUpdated(old interface{}, new interface{}) {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	for _, h := range b.handlers {
		if h.UpdateFunc != nil {
			h.UpdateFunc(old, new)
		}
	}
}

func (b *baseDomain) resourceDeleted(del interface{}) {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	for _, h := range b.handlers {
		if h.DeleteFunc != nil {
			h.DeleteFunc(del)
		}
	}
}

func (b *baseDomain) addHandler(h *ModificationHandler) {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	b.handlers = append(b.handlers, h)
}
