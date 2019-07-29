// +build basic

package sample

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestPassTag(t *testing.T) {
	g := NewWithT(t)

	logrus.Infof("Passed test")
}

func TestFailTag(t *testing.T) {
	g := NewWithT(t)

	logrus.Infof("Failed test")

	g.Expect("fail").To(Equal("success"))
}

func TestTimeoutTag(t *testing.T) {
	g := NewWithT(t)

	logrus.Infof("test timeout for 5 seconds")
	<-time.After(5 * time.Second)
}
