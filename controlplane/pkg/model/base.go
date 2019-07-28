package model

import (
	"sync"

	"github.com/sirupsen/logrus"
)

// ModificationHandler aggregates handlers for particular events
type ModificationHandler struct {
	AddFunc    func(new interface{})
	UpdateFunc func(old interface{}, new interface{})
	DeleteFunc func(del interface{})
}

type cloneable interface {
	clone() cloneable
}

type baseDomain struct {
	mtx      sync.RWMutex
	handlers []*ModificationHandler
	innerMap map[string]cloneable
}

func newBase() baseDomain {
	return baseDomain{
		innerMap: map[string]cloneable{},
	}
}

func (b *baseDomain) load(key string) (interface{}, bool) {
	b.mtx.RLock()
	defer b.mtx.RUnlock()

	v, exist := b.innerMap[key]
	if !exist {
		return nil, false
	}
	return v.clone(), true
}

func (b *baseDomain) store(key string, value cloneable) {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	old, exist := b.innerMap[key]
	if !exist {
		b.innerMap[key] = value.clone()
		b.resourceAdded(value)
		return
	}

	b.innerMap[key] = value.clone()
	b.resourceUpdated(old, value)
}

func (b *baseDomain) delete(key string) {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	v, exist := b.innerMap[key]
	if !exist {
		return
	}
	delete(b.innerMap, key)
	b.resourceDeleted(v)
}

func (b *baseDomain) applyChanges(key string, changeFunc func(interface{})) interface{} {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	old, exist := b.innerMap[key]
	if !exist {
		return nil
	}

	upd := old.clone()
	changeFunc(upd)

	b.innerMap[key] = upd.clone()
	b.resourceUpdated(old, upd)
	return upd
}

func (b *baseDomain) kvRange(f func(key string, v interface{}) bool) {
	b.mtx.RLock()
	defer b.mtx.RUnlock()

	for k, v := range b.innerMap {
		if !f(k, v.clone()) {
			return
		}
	}
}

func (b *baseDomain) resourceAdded(new cloneable) {
	logrus.Infof("resourceAdded started: %v", new)

	for _, h := range b.handlers {
		if h.AddFunc != nil {
			go h.AddFunc(new.clone())
		}
	}
	logrus.Infof("resourceAdded finished: %v", new)
}

func (b *baseDomain) resourceUpdated(old, new cloneable) {
	logrus.Infof("resourceUpdated started: %v", new)

	for _, h := range b.handlers {
		if h.UpdateFunc != nil {
			go h.UpdateFunc(old.clone(), new.clone())
		}
	}
	logrus.Infof("resourceUpdated finished: %v", new)
}

func (b *baseDomain) resourceDeleted(del cloneable) {
	logrus.Infof("resourceDeleted started: %v", del)

	for _, h := range b.handlers {
		if h.DeleteFunc != nil {
			go h.DeleteFunc(del.clone())
		}
	}
	logrus.Infof("resourceDeleted finished: %v", del)
}

func (b *baseDomain) addHandler(h *ModificationHandler) func() {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	b.handlers = append(b.handlers, h)

	for _, v := range b.innerMap {
		b.resourceAdded(v)
	}

	return func() {
		b.deleteHandler(h)
	}
}

func (b *baseDomain) deleteHandler(h *ModificationHandler) {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	for i := 0; i < len(b.handlers); i++ {
		if h == b.handlers[i] {
			b.handlers = append(b.handlers[:i], b.handlers[i+1:]...)
			return
		}
	}
}
