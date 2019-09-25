package model

import (
	"context"
	"sync"
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

type testResource struct {
	value string
}

func (r *testResource) clone() cloneable {
	return &testResource{
		value: r.value,
	}
}

func TestModificationHandler(t *testing.T) {
	g := NewWithT(t)

	bd := baseDomain{}
	resource := &testResource{"test"}
	updResource := &testResource{"updated"}

	amountHandlers := 5

	var wg sync.WaitGroup
	for i := 0; i < amountHandlers; i++ {
		wg.Add(3)
		bd.addHandler(&ModificationHandler{
			AddFunc: func(ctx context.Context, new interface{}) {
				defer wg.Done()
				g.Expect(new.(*testResource).value).To(Equal(resource.value))
			},
			UpdateFunc: func(ctx context.Context, old interface{}, new interface{}) {
				defer wg.Done()
				g.Expect(old.(*testResource).value).To(Equal(resource.value))
				g.Expect(new.(*testResource).value).To(Equal(updResource.value))
			},
			DeleteFunc: func(ctx context.Context, del interface{}) {
				defer wg.Done()
				g.Expect(del.(*testResource).value).To(Equal(resource.value))
			},
		})
	}

	bd.resourceAdded(context.Background(), resource)
	bd.resourceUpdated(context.Background(), resource, updResource)
	bd.resourceDeleted(context.Background(), resource)
	doneCh := make(chan struct{})

	go func() {
		wg.Wait()
		close(doneCh)
	}()

	select {
	case <-time.After(5 * time.Second):
		t.Fatal("not all listeners have been emitted ")
	case <-doneCh:
		return
	}
}
