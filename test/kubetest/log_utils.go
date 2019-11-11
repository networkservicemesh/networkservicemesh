package kubetest

import (
	"path"
	"testing"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest/artifact"
)

// MakeLogsSnapshot prints logs from containers in case of fail/panic or enabled logging in file
func MakeLogsSnapshot(k8s *K8s, t *testing.T) {
	makeLogsSnapShot := func() {
		if !k8s.artifactsConfig.SaveInAnyCase() {
			k8s.resourcesBehaviour = DefaultClear
		}
		c := artifact.UpdateConfigOutputPath(k8s.artifactsConfig, path.Join(k8s.artifactsConfig.OutputPath(), t.Name()))
		m := artifact.NewManager(c, artifact.DefaultPresenterFactory(), []artifact.Finder{
			NewK8sLogFinder(k8s),
			NewJaegerTracesFinder(k8s),
		}, nil)
		m.ProcessArtifacts()
	}

	if exception := recover(); exception != nil {
		makeLogsSnapShot()
		panic(exception)
	} else if t.Failed() || k8s.artifactsConfig.SaveInAnyCase() {
		makeLogsSnapShot()
	}
}
