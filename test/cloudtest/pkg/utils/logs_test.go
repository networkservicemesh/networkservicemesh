package utils

import (
	"bytes"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	"github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"testing"
)

func TestLogsParse(t *testing.T) {
	gomega.RegisterTestingT(t)
	buf := bytes.NewBuffer([]byte{})
	logrus.SetOutput(buf)
	for i := 0; i <= 1e2; i++ {
		logrus.Info(i)
	}
	logs := bytes.NewBuffer([]byte{})
	logrus.SetOutput(logs)
	kubetest.LogTransaction("T1", buf.String())
	kubetest.LogTransaction("T2", buf.String())
	logsOfPods := CollectLogs(logs.String())
	gomega.Expect(2).Should(gomega.Equal(len(logsOfPods)))
	gomega.Expect(logsOfPods[0].Logs).Should(gomega.Equal(buf.String()))
	gomega.Expect(logsOfPods[0].ContainerName).Should(gomega.Equal("T1"))
	gomega.Expect(logsOfPods[1].Logs).Should(gomega.Equal(buf.String()))
	gomega.Expect(logsOfPods[1].ContainerName).Should(gomega.Equal("T2"))
}
