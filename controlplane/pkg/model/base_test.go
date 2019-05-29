package model

import (
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"testing"
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
	RegisterTestingT(t)

	bd := baseDomain{}
	resource := &testResource{"test"}
	updResource := &testResource{"updated"}

	amountHandlers := 5
	addCalledTimes := make([]bool, amountHandlers)
	updateCalledTimes := make([]bool, amountHandlers)
	deleteCalledTimes := make([]bool, amountHandlers)

	for i := 0; i < amountHandlers; i++ {
		logrus.Info(addCalledTimes[i])
		addPtr := &addCalledTimes[i]
		updPtr := &updateCalledTimes[i]
		delPtr := &deleteCalledTimes[i]
		bd.addHandler(&ModificationHandler{
			AddFunc: func(new interface{}) {
				*addPtr = true
				Expect(new.(*testResource).value).To(Equal(resource.value))
			},
			UpdateFunc: func(old interface{}, new interface{}) {
				*updPtr = true
				Expect(old.(*testResource).value).To(Equal(resource.value))
				Expect(new.(*testResource).value).To(Equal(updResource.value))
			},
			DeleteFunc: func(del interface{}) {
				*delPtr = true
				Expect(del.(*testResource).value).To(Equal(resource.value))
			},
		})
	}

	bd.resourceAdded(resource)
	bd.resourceUpdated(resource, updResource)
	bd.resourceDeleted(resource)

	for i := 0; i < amountHandlers; i++ {
		Expect(addCalledTimes[i]).To(BeTrue())
		Expect(updateCalledTimes[i]).To(BeTrue())
		Expect(deleteCalledTimes[i]).To(BeTrue())
	}
}
