package errtools

import (
	"errors"
	"strings"
	"testing"

	"github.com/onsi/gomega"
)

func TestCollect(t *testing.T) {
	assert := gomega.NewGomegaWithT(t)
	actual := Collect(nil)
	assert.Expect(actual).Should(gomega.BeNil())
	actual = Collect(nil, nil, nil, nil)
	assert.Expect(actual).Should(gomega.BeNil())
	err1 := errors.New("one")
	actual = Collect(err1)
	assert.Expect(actual).Should(gomega.Equal(err1))
	err2 := errors.New("two")
	actual = Collect(err1, nil, nil, err2)
	assert.Expect(actual).ShouldNot(gomega.BeNil())
	assert.Expect(strings.Contains(actual.Error(), err1.Error()) && strings.Contains(actual.Error(), err2.Error())).Should(gomega.BeTrue())
}
