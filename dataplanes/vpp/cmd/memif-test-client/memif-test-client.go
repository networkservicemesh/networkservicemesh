package main

import (
	"context"
	"flag"
	"github.com/ligato/networkservicemesh/dataplanes/vpp/pkg/nsmutils"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/common"
	dataplaneapi "github.com/ligato/networkservicemesh/pkg/nsm/apis/dataplane"
	"github.com/ligato/networkservicemesh/pkg/tools"
	"github.com/sirupsen/logrus"
	"os"
)

const (
	dataplane = "/var/lib/networkservicemesh/nsm-vpp.dataplane.sock"
)

var (
	srcPodDirectory = flag.String("src-pod-dir", "", "Name of the source pod")
	dstPodDirectory = flag.String("dst-pod-dir", "", "Name of the destination pod")
)

type dataplaneClientTest struct {
	testName         string
	localSource      *common.LocalMechanism
	localDestination *dataplaneapi.Connection_Local
	shouldFail       bool
}

func main() {
	flag.Parse()

	if *srcPodDirectory == "" || *dstPodDirectory == "" {
		logrus.Fatal("Both source and destination PODs' directories must be specified, exitting...")
	}
	if _, err := os.Stat(dataplane); err != nil {
		logrus.Fatalf("nsm-vpp-dataplane: failure to access nsm socket at %s with error: %+v, exiting...", dataplane, err)
	}
	conn, err := tools.SocketOperationCheck(dataplane)
	if err != nil {
		logrus.Fatalf("nsm-vpp-dataplane: failure to communicate with the socket %s with error: %+v", dataplane, err)
	}
	defer conn.Close()
	logrus.Infof("nsm-vpp-dataplane: connection to dataplane registrar socket %s succeeded.", dataplane)

	dataplaneClient := dataplaneapi.NewDataplaneOperationsClient(conn)

	tests := []dataplaneClientTest{
		{
			testName: "all good",
			localSource: &common.LocalMechanism{
				Type: common.LocalMechanismType_MEM_INTERFACE,
				Parameters: map[string]string{
					nsmutils.NSMSocketFile:      "src.sock",
					nsmutils.NSMMaster:          "true",
					nsmutils.NSMPerPodDirectory: *srcPodDirectory,
				},
			},
			localDestination: &dataplaneapi.Connection_Local{
				Local: &common.LocalMechanism{
					Type: common.LocalMechanismType_MEM_INTERFACE,
					Parameters: map[string]string{
						nsmutils.NSMSocketFile:      "dst.sock",
						nsmutils.NSMSlave:           "true",
						nsmutils.NSMPerPodDirectory: *dstPodDirectory,
					},
				},
			},
			shouldFail: false,
		},
	}

	for _, test := range tests {
		logrus.Infof("Running test: %s", test.testName)
		_, err := dataplaneClient.ConnectRequest(context.Background(), &dataplaneapi.Connection{
			LocalSource: test.localSource,
			Destination: test.localDestination,
		})
		if err != nil {
			if !test.shouldFail {
				logrus.Fatalf("Test %s failed with error: %+v but should not.", test.testName, err)
			}
		}
		if err == nil {
			if test.shouldFail {
				logrus.Fatalf("Test %s did not fail but should.", test.testName)
			}
		}
		logrus.Infof("Test %s has finished with expected result.", test.testName)
	}
}
