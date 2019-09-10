package main

import "testing"

func TestAdmissionWebhookGoals(t *testing.T) {
	g := admissionWebhookGoals{}
	if g.IsComplete() {
		t.FailNow()
	}
	g.SetServerStarted()
	if g.IsComplete() {
		t.FailNow()
	}
	g.SetKeyPairLoaded()
	if !g.IsComplete() {
		t.FailNow()
	}
}
