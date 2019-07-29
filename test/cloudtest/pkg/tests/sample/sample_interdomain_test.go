// +build interdomain

package sample

import (
	"os"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestInterDomainPass(t *testing.T) {
	g := NewWithT(t)

	logrus.Infof("Passed test")
}

func TestInterdomainCheck(t *testing.T) {
	g := NewWithT(t)

	g.Expect(len(os.Getenv("CFG1")) != 0).To(Equal(true))
	g.Expect(len(os.Getenv("CFG1")) != 0).To(Equal(true))
}
func TestInterdomainFail(t *testing.T) {
	g := NewWithT(t)

	logrus.Infof("Failed test")

	g.Expect("fail").To(Equal("success"))
}
