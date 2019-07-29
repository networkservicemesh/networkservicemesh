package sample

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestPass(t *testing.T) {
	logrus.Infof("Passed test")
}

func TestFail(t *testing.T) {
	g := NewWithT(t)

	logrus.Infof("Failed test")

	g.Expect("fail").To(Equal("success"))
}

func TestTimeout(t *testing.T) {
	logrus.Infof("test timeout for 5 seconds")
	<-time.After(5 * time.Second)
}
