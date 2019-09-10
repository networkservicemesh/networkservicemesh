package main

import "testing"

func TestNsmdProbeGoals(t *testing.T) {
	g := nsmdProbeGoals{}
	if g.IsComplete() {
		t.FailNow()
	}
	g.SetServerAPIReady()
	if g.IsComplete() {
		t.FailNow()
	}
	g.SetPublicListenerReady()
	if g.IsComplete() {
		t.FailNow()
	}
	g.SetNsmServerReady()
	if g.IsComplete() {
		t.FailNow()
	}
	g.SetDataplaneServerReady()
	if !g.IsComplete() {
		t.FailNow()
	}
}
