package model

import (
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"testing"
)

func TestModificationHandler(t *testing.T) {
	RegisterTestingT(t)

	bd := baseDomain{}
	resource := "test"
	updResource := "updated"

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
				Expect(new.(string)).To(Equal(resource))
			},
			UpdateFunc: func(old interface{}, new interface{}) {
				*updPtr = true
				Expect(old.(string)).To(Equal(resource))
				Expect(new.(string)).To(Equal(updResource))
			},
			DeleteFunc: func(del interface{}) {
				*delPtr = true
				Expect(del.(string)).To(Equal(resource))
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
