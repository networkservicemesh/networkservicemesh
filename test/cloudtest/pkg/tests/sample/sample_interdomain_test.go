// +build interdomain

package sample

import (
	"os"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestInterDomainPass(t *testing.T) {
	RegisterTestingT(t)

	logrus.Infof("Passed test")
}

func TestInterdomainCheck(t *testing.T) {
	RegisterTestingT(t)

	Expect(len(os.Getenv("CFG1")) != 0).To(Equal(true))
	Expect(len(os.Getenv("CFG1")) != 0).To(Equal(true))
}
func TestInterdomainFail(t *testing.T) {
	RegisterTestingT(t)

	logrus.Infof("Failed test")

	Expect("fail").To(Equal("success"))
}
