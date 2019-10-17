package common

import "testing"

func TestDataplaneProbeGoals(t *testing.T) {
	g := DataplaneProbeGoals{}
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
