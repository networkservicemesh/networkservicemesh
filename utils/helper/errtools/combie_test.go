package errtools

import (
	"errors"
	"testing"

	"github.com/onsi/gomega"
)

func TestCombine(t *testing.T) {
	assert := gomega.NewWithT(t)
	err1 := errors.New("err1")
	err2 := errors.New("err2")
	err3 := Combine(err1, nil, nil, err2)
	expected := `1. err1
2. err2`
	assert.Expect(err3.Error()).Should(gomega.Equal(expected))
}
func TestNilCombine(t *testing.T) {
	assert := gomega.NewWithT(t)
	err := Combine(nil)
	assert.Expect(err).Should(gomega.BeNil())
	err = Combine(nil, nil, nil)
	assert.Expect(err).Should(gomega.BeNil())
}

func TestSingleCombine(t *testing.T) {
	assert := gomega.NewWithT(t)
	err1 := errors.New("err1")
	err2 := Combine(err1)
	assert.Expect(err1).Should(gomega.Equal(err2))
}
