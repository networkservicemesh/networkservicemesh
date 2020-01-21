package main

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/forwarder/pkg/common"
	"github.com/networkservicemesh/networkservicemesh/forwarder/sriov-forwarder/pkg/sriovforwarder"
	"github.com/networkservicemesh/networkservicemesh/pkg/probes"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools/jaeger"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools/spanhelper"
)

func main() {

	logrus.Info("Starting nsm-sriov-forwarder...")

	closer := jaeger.InitJaeger("nsm-sriov-forwarder")
	defer func() { _ = closer.Close() }()

	span := spanhelper.FromContext(context.Background(), "Start.SRIOVForwarder.Dataplane")
	defer span.Finish()

	c := tools.NewOSSignalChannel()

	dataplaneGoals := &common.ForwarderProbeGoals{}
	dataplaneProbes := probes.New("SRIOV-based forwarding dataplane liveness/readiness healthcheck", dataplaneGoals)
	dataplaneProbes.BeginHealthCheck()

	forwarder := sriovforwarder.CreateSRIOVForwarder()

	registration := common.CreateForwarder(span.Context(), forwarder, dataplaneGoals)

	for range c {
		logrus.Info("Closing Dataplane Registration")
		registration.Close()
	}
}
