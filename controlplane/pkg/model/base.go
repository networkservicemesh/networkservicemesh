package model

import (
	"context"
	"sync"

	"github.com/sirupsen/logrus"
)

// ModificationHandler aggregates handlers for particular events
type ModificationHandler struct {
	AddFunc    func(ctx context.Context, new interface{})
	UpdateFunc func(ctx context.Context, old interface{}, new interface{})
	DeleteFunc func(ctx context.Context, del interface{})
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

func (b *baseDomain) store(ctx context.Context, key string, value cloneable) {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	old, exist := b.innerMap[key]
	if !exist {
		b.innerMap[key] = value.clone()
		b.resourceAdded(ctx, value)
		return
	}

	b.innerMap[key] = value.clone()
	b.resourceUpdated(ctx, old, value)
}

func (b *baseDomain) delete(ctx context.Context, key string) {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	v, exist := b.innerMap[key]
	if !exist {
		return
	}
	delete(b.innerMap, key)
	b.resourceDeleted(ctx, v)
}

func (b *baseDomain) applyChanges(ctx context.Context, key string, changeFunc func(interface{})) interface{} {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	old, exist := b.innerMap[key]
	if !exist {
		return nil
	}

	upd := old.clone()
	changeFunc(upd)

	b.innerMap[key] = upd.clone()
	b.resourceUpdated(ctx, old, upd)
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

func (b *baseDomain) resourceAdded(ctx context.Context, new cloneable) {
	logrus.Infof("resourceAdded started: %v", new)

	for _, h := range b.handlers {
		if h.AddFunc != nil {
			go h.AddFunc(ctx, new.clone())
		}
	}
	logrus.Infof("resourceAdded finished: %v", new)
}

func (b *baseDomain) resourceUpdated(ctx context.Context, old, new cloneable) {
	logrus.Infof("resourceUpdated started: %v", new)

	for _, h := range b.handlers {
		if h.UpdateFunc != nil {
			go h.UpdateFunc(ctx, old.clone(), new.clone())
		}
	}
	logrus.Infof("resourceUpdated finished: %v", new)
}

func (b *baseDomain) resourceDeleted(ctx context.Context, del cloneable) {
	logrus.Infof("resourceDeleted started: %v", del)

	for _, h := range b.handlers {
		if h.DeleteFunc != nil {
			go h.DeleteFunc(ctx, del.clone())
		}
	}
	logrus.Infof("resourceDeleted finished: %v", del)
}

func (b *baseDomain) addHandler(h *ModificationHandler) func() {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	b.handlers = append(b.handlers, h)

	for _, v := range b.innerMap {
		b.resourceAdded(context.Background(), v)
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
