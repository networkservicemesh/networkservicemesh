package model

import (
	. "github.com/onsi/gomega"
	"testing"
)

func TestModelAddRemove(t *testing.T) {
	RegisterTestingT(t)

	model := NewModel()

	model.AddDataplane( &Dataplane{
		RegisteredName: "test_name",
		SocketLocation: "location",
	})

	Expect(model.GetDataplane("test_name").RegisteredName).To(Equal("test_name"))

	model.DeleteDataplane("test_name")

	Expect(model.GetDataplane("test_name")).To(BeNil())
}

func TestModelSelectDataplane(t *testing.T) {
	RegisterTestingT(t)

	model := NewModel()

	model.AddDataplane( &Dataplane{
		RegisteredName: "test_name",
		SocketLocation: "location",
	})
	dp, err := model.SelectDataplane()
	Expect(dp.RegisteredName).To(Equal("test_name"))
	Expect(err).To(BeNil())
}
func TestModelSelectDataplaneNone(t *testing.T) {
	RegisterTestingT(t)

	model := NewModel()

	dp, err := model.SelectDataplane()
	Expect(dp).To(BeNil())
	Expect(err.Error()).To(Equal("no dataplanes registered"))
}

