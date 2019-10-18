package common

import "testing"

func TestForwarderProbeGoals(t *testing.T) {

	g := ForwarderProbeGoals{}
	if g.IsComplete() {
		t.FailNow()
	}
	g.SetNewEgressIFReady()
	if g.IsComplete() {
		t.FailNow()
	}
	g.SetSrcIPReady()
	if g.IsComplete() {
		t.FailNow()
	}
	g.SetSocketCleanReady()
	if g.IsComplete() {
		t.FailNow()
	}
	g.SetValidIPReady()
	if g.IsComplete() {
		t.FailNow()
	}
	g.SetSocketListenReady()
	if !g.IsComplete() {
		t.FailNow()
	}
}
