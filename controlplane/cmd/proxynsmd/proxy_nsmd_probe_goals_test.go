package main

import "testing"

func TestNsmdProbeGoals(t *testing.T) {
	g := proxyNsmdProbeGoals{}
	if g.IsComplete() {
		t.FailNow()
	}
	g.SetServerAPIReady()
	if g.IsComplete() {
		t.FailNow()
	}
	g.SetPublicListenerReady()
	if !g.IsComplete() {
		t.FailNow()
	}
}
