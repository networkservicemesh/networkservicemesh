// Copyright (c) 2018 Cisco and/or its affiliates.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ligato/networkservicemesh/dataplanes/vpp/pkg/nsmutils"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/common"
	dataplaneapi "github.com/ligato/networkservicemesh/pkg/nsm/apis/dataplane"
	dataplaneregistrarapi "github.com/ligato/networkservicemesh/pkg/nsm/apis/dataplaneregistrar"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/testdataplane"
	"github.com/ligato/networkservicemesh/pkg/tools"
	"github.com/ligato/networkservicemesh/plugins/dataplaneregistrar"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	dataplaneSocket           = "/var/lib/networkservicemesh/dataplane.sock"
	interfaceNameMaxLength    = 15
	registrationRetryInterval = 30
)

var (
	dataplane       = flag.String("dataplane-socket", dataplaneSocket, "Location of the dataplane gRPC socket")
	kubeconfig      = flag.String("kubeconfig", "", "Absolute path to the kubeconfig file. Either this or master needs to be set if the provisioner is being run out of cluster.")
	registrarSocket = path.Join(dataplaneregistrar.DataplaneRegistrarSocketBaseDir, dataplaneregistrar.DataplaneRegistrarSocket)
)

// DataplaneController keeps k8s client and gRPC server
type DataplaneController struct {
	k8s              *kubernetes.Clientset
	dataplaneServer  *grpc.Server
	remoteMechanisms []*common.RemoteMechanism
	localMechanisms  []*common.LocalMechanism
	updateCh         chan Update
}

// Update is a message used to communicate any changes in operational parameters and constraints
type Update struct {
	remoteMechanisms []*common.RemoteMechanism
	localMechanisms  []*common.LocalMechanism
}

// livenessMonitor is a stream initiated by NSM to inform the dataplane that NSM is still alive and
// no re-registration is required. Detection a failure on this "channel" will mean
// that NSM is gone and the dataplane needs to start re-registration logic.
func livenessMonitor(registrationConnection dataplaneregistrarapi.DataplaneRegistrationClient) {
	stream, err := registrationConnection.RequestLiveness(context.Background())
	if err != nil {
		logrus.Errorf("test-dataplane: fail to create liveness grpc channel with NSM with error: %s, grpc code: %+v", err.Error(), status.Convert(err).Code())
		return
	}
	for {
		err := stream.RecvMsg(&common.Empty{})
		if err != nil {
			logrus.Errorf("test-dataplane: fail to receive from liveness grpc channel with error: %s, grpc code: %+v", err.Error(), status.Convert(err).Code())
			return
		}
	}
}

// UpdateDataplane implements method of dataplane interface, which is informing NSM of any changes
// to operational prameters or constraints
func (d DataplaneController) MonitorMechanisms(empty *common.Empty, updateSrv dataplaneapi.DataplaneOperations_MonitorMechanismsServer) error {
	logrus.Infof("Update dataplane was called")
	if err := updateSrv.Send(&dataplaneapi.MechanismUpdate{
		RemoteMechanisms: d.remoteMechanisms,
		LocalMechanisms:  d.localMechanisms,
	}); err != nil {
		logrus.Errorf("test-dataplane: Deteced error %s, grpc code: %+v on grpc channel", err.Error(), status.Convert(err).Code())
		return nil
	}
	for {
		select {
		// Waiting for any updates which might occur during a life of dataplane module and communicating
		// them back to NSM.
		case update := <-d.updateCh:
			d.localMechanisms = update.localMechanisms
			d.remoteMechanisms = update.remoteMechanisms
			if err := updateSrv.Send(&dataplaneapi.MechanismUpdate{
				RemoteMechanisms: update.remoteMechanisms,
				LocalMechanisms:  update.localMechanisms,
			}); err != nil {
				logrus.Errorf("test-dataplane: Deteced error %s, grpc code: %+v on grpc channel", err.Error(), status.Convert(err).Code())
				return nil
			}
		}
	}
}

// ConnectRequest implements method of dataplane interface, NSM sends ConnectRequest to the dataplane of behalf
// of NSM Client or NSE.
func (d DataplaneController) ConnectRequest(ctx context.Context, req *dataplaneapi.Connection) (*dataplaneapi.Reply, error) {
	logrus.Infof("ConnectRequest was called")

	source := req.LocalSource.GetParameters()
	srcNS := source[nsmutils.NSMkeyNamespace]
	pid1, err := strconv.Atoi(strings.Split(srcNS, ":")[1])
	if err != nil {
		return nil, fmt.Errorf("bad namespace %v", srcNS)
	}

	destinationLocal := req.Destination.(*dataplaneapi.Connection_Local)
	destination := destinationLocal.Local.GetParameters()
	dstNS := destination[nsmutils.NSMkeyNamespace]
	pid2, err := strconv.Atoi(strings.Split(dstNS, ":")[1])
	if err != nil {
		return nil, fmt.Errorf("bad namespace %v", dstNS)
	}

	logrus.Infof("connecting processes: %d %d", pid1, pid2)

	err = connectProcesses(pid1, pid2)
	if err != nil {
		return &dataplaneapi.Reply{
			Success:      false,
			ConnectionId: err.Error(),
		}, err
	}

	connID := fmt.Sprintf("%d-%d", pid1, pid2)

	return &dataplaneapi.Reply{
		Success:      true,
		ConnectionId: connID,
	}, nil
}

// DisconnectRequest implements method of dataplane interface, NSM sends ConnectRequest to the dataplane of behalf
// of NSM Client or NSE.
func (d DataplaneController) DisconnectRequest(ctx context.Context, req *dataplaneapi.Connection) (*dataplaneapi.Reply, error) {
	logrus.Infof("DisconnectRequest was called")
	return nil, fmt.Errorf("not implemented")
}

// RequestDeleteConnect implements method for testdataplane proto
func (d DataplaneController) RequestDeleteConnect(ctx context.Context, in *testdataplane.DeleteConnectRequest) (*testdataplane.DeleteConnectReply, error) {
	logrus.Infof("Request Delete Connect received for pod: %s/%s pod type: %v", in.Pod.Metadata.Namespace, in.Pod.Metadata.Name, in.PodType)
	podName := in.Pod.Metadata.Name
	if podName == "" {
		logrus.Error("test-dataplane: missing required pod name")
		return &testdataplane.DeleteConnectReply{
			Deleted:     false,
			DeleteError: fmt.Sprint("missing required name for pod"),
		}, status.Error(codes.NotFound, "missing required pod name")
	}
	podNamespace := "default"
	if in.Pod.Metadata.Namespace != "" {
		podNamespace = in.Pod.Metadata.Namespace
	}

	switch in.PodType {
	case testdataplane.NSMPodType_NSE:
	case testdataplane.NSMPodType_NSMCLIENT:
	default:
		logrus.Error("test-dataplane: invalid pod type")
		return &testdataplane.DeleteConnectReply{
			Deleted:     false,
			DeleteError: fmt.Sprintf("invalid pod type detected for pod %s/%s", podNamespace, podName),
		}, status.Error(codes.Aborted, "nvalid pod type")
	}
	if err := deleteLink(d.k8s, podName, podNamespace, in.PodType); err != nil {
		logrus.Errorf("Request Delete Connect failed for pod: %s/%s pod type: %v with error: %+v", in.Pod.Metadata.Namespace, in.Pod.Metadata.Name, in.PodType, err)
		return &testdataplane.DeleteConnectReply{
			Deleted:     false,
			DeleteError: fmt.Sprintf("failed to delete interface(s) for pod %s/%s with error: %+v", podNamespace, podName, err),
		}, status.Error(codes.Aborted, "failed to delete interface(s) for pod")
	}
	logrus.Infof("Request Delete Connect succeeded for pod: %s/%s pod type: %v", in.Pod.Metadata.Namespace, in.Pod.Metadata.Name, in.PodType)
	return &testdataplane.DeleteConnectReply{
		Deleted: true,
	}, nil
}

// RequestBuildConnect implements method for testdataplane proto
func (d DataplaneController) RequestBuildConnect(ctx context.Context, in *testdataplane.BuildConnectRequest) (*testdataplane.BuildConnectReply, error) {

	podName1 := in.SourcePod.Metadata.Name
	if podName1 == "" {
		logrus.Error("test-dataplane: missing required pod name")
		return &testdataplane.BuildConnectReply{
			Built:      false,
			BuildError: fmt.Sprint("missing required name for pod 1"),
		}, status.Error(codes.NotFound, "missing required pod name")
	}
	podNamespace1 := "default"
	if in.SourcePod.Metadata.Namespace != "" {
		podNamespace1 = in.SourcePod.Metadata.Namespace
	}

	podName2 := in.DestinationPod.Metadata.Name
	if podName2 == "" {
		logrus.Error("test-dataplane: missing required pod name")
		return &testdataplane.BuildConnectReply{
			Built:      false,
			BuildError: fmt.Sprint("missing required name for pod 2"),
		}, status.Error(codes.NotFound, "missing required pod name")
	}
	podNamespace2 := "default"
	if in.DestinationPod.Metadata.Namespace != "" {
		podNamespace2 = in.DestinationPod.Metadata.Namespace
	}

	logrus.Infof("test-dataplane: attempting to interconnect pods %s/%s and %s/%s",
		podNamespace1,
		podName1,
		podNamespace2,
		podName2)
	// TODO (sbezverk) Add ip address check
	if err := connectPods(d.k8s, podName1, podName2, podNamespace1, podNamespace2); err != nil {
		logrus.Errorf("test-dataplane: failed to interconnect pods %s/%s and %s/%s with error: %+v",
			podNamespace1,
			podName1,
			podNamespace2,
			podName2,
			err)
		return &testdataplane.BuildConnectReply{
			Built: false,
			BuildError: fmt.Sprintf("test-dataplane: failed to interconnect pods %s/%s and %s/%s with error: %+v",
				podNamespace1,
				podName1,
				podNamespace2,
				podName2,
				err),
		}, status.Error(codes.Aborted, "failure to interconnect pods")
	}

	return &testdataplane.BuildConnectReply{
		Built: true,
	}, nil
}

func buildClient() (*kubernetes.Clientset, error) {
	var config *rest.Config
	var err error

	kubeconfigEnv := os.Getenv("KUBECONFIG")

	if kubeconfigEnv != "" {
		kubeconfig = &kubeconfigEnv
	}

	if *kubeconfig != "" {
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
	} else {
		config, err = rest.InClusterConfig()
	}
	if err != nil {
		return nil, err
	}
	k8s, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return k8s, nil
}

func getContainerID(k8s *kubernetes.Clientset, pn, ns string) (string, error) {
	pl, err := k8s.CoreV1().Pods(ns).List(metav1.ListOptions{})
	if err != nil {
		return "", err
	}

	for _, p := range pl.Items {
		if strings.HasPrefix(p.ObjectMeta.Name, pn) {
			// Two cases main container is in Running state and in Pending
			// Pending can inidcate that Init container is still running
			if p.Status.Phase == v1.PodRunning {
				return strings.Split(p.Status.ContainerStatuses[0].ContainerID, "://")[1][:12], nil
			}
			if p.Status.Phase == v1.PodPending {
				// Check if we have Init containers
				if p.Status.InitContainerStatuses != nil {
					for _, i := range p.Status.InitContainerStatuses {
						if i.State.Running != nil {
							return strings.Split(i.ContainerID, "://")[1][:12], nil
						}
					}
				}
			}
			return "", fmt.Errorf("test-dataplane: none of containers of pod %s/%s is in running state", p.ObjectMeta.Namespace, p.ObjectMeta.Name)
		}
	}

	return "", fmt.Errorf("test-dataplane: pod %s/%s not found", ns, pn)
}

func deleteVethInterface(ns netns.NsHandle, interfacePrefix string) error {
	nsOrigin, err := netns.Get()
	if err != nil {
		return fmt.Errorf("failed to get current process namespace")
	}
	defer netns.Set(nsOrigin)

	if err := netns.Set(ns); err != nil {
		return fmt.Errorf("failed to switch to namespace %s with error: %+v", ns, err)
	}
	namespaceHandle, err := netlink.NewHandleAt(ns)
	if err != nil {
		return fmt.Errorf("failure to get pod's handle with error: %+v", err)
	}

	// Getting list of all interfaces in pod's namespace
	links, err := namespaceHandle.LinkList()
	if err != nil {
		return fmt.Errorf("failure to get pod's interfaces with error: %+v", err)
	}

	// Now walk the list of discovered interfaces and shut down and delete all links
	// with matching interfacePrefix.
	// interfacePrefix can have two values "nsm" or "nse". All NSM injected interfaces
	// to NSM client pod will have prefix "nse" and all injected interfaces of NSE pod
	// will have prefix "nsm".
	for _, link := range links {
		if strings.HasPrefix(link.Attrs().Name, interfacePrefix) {
			// Found NSM related interface and in best effort mode shutting it down
			// and then delete it
			_ = netlink.LinkSetDown(link)
			_ = namespaceHandle.LinkDel(link)
		}
	}

	return nil
}

func setVethPair(ns1, ns2 netns.NsHandle, p1, p2 string) error {
	ns, err := netns.Get()
	if err != nil {
		return fmt.Errorf("failed to get current process namespace")
	}
	defer netns.Set(ns)

	linkAttr := netlink.NewLinkAttrs()
	linkAttr.Name = p2
	veth := &netlink.Veth{
		LinkAttrs: linkAttr,
		PeerName:  p1,
	}
	if err := netns.Set(ns1); err != nil {
		return fmt.Errorf("failed to switch to namespace %s with error: %+v", ns1, err)
	}
	namespaceHandle, err := netlink.NewHandleAt(ns1)
	if err != nil {
		return fmt.Errorf("failure to get pod's handle with error: %+v", err)
	}
	// Debugging situation when we try to add already existing link
	links, err := namespaceHandle.LinkList()
	if err != nil {
		logrus.Errorf("failure to get a list of links from the namespace %s with error: %+v", ns1.String(), err)
		return fmt.Errorf("failure to get a list of links from the namespace %s with error: %+v", ns1.String(), err)
	}
	logrus.Printf("Found already existing interfaces:")
	found := false
	for _, link := range links {
		addrs, err := namespaceHandle.AddrList(link, 0)
		if err != nil {
			logrus.Printf("failed to addresses for interface: %s with error: %v", link.Attrs().Name, err)
		}
		logrus.Printf("Name: %s Type: %s Addresses: %+v", link.Attrs().Name, link.Type(), addrs)
		// Check if there is already interface with the same name, if there is, then skip crreating it
		if link.Attrs().Name == veth.LinkAttrs.Name {
			logrus.Printf("Interface with %s already exist, skip creating it.", veth.LinkAttrs.Name)
			found = true
		}
	}
	if !found {
		// Interface has not been found, safe to create it.
		if err := namespaceHandle.LinkAdd(veth); err != nil {
			return fmt.Errorf("failure to add veth to pod with error: %+v", err)
		}
		// Adding a small timeout to let interface add to complete
		time.Sleep(30 * time.Second)
	}

	link, err := netlink.LinkByName(p2)
	if err != nil {
		return fmt.Errorf("failure to get pod's interface by name with error: %+v", err)
	}
	if _, ok := link.(*netlink.Veth); !ok {
		return fmt.Errorf("failure, got unexpected interface type: %+v", reflect.TypeOf(link))
	}
	if err := netlink.LinkSetUp(link); err != nil {
		return fmt.Errorf("failure setting link %s up with error: %+v", link.Attrs().Name, err)
	}
	peer, err := netlink.LinkByName(p1)
	if err != nil {
		return fmt.Errorf("failure to get pod's interface by name with error: %+v", err)
	}
	if _, ok := peer.(*netlink.Veth); !ok {
		return fmt.Errorf("failure, got unexpected interface type: %+v", reflect.TypeOf(link))
	}
	// Moving peer's interface into peer's namespace
	if err := netlink.LinkSetNsFd(peer, int(ns2)); err != nil {
		return fmt.Errorf("failure to get place veth into peer's pod with error: %+v", err)
	}
	// Switching to peer's namespace
	if err := netns.Set(ns2); err != nil {
		return fmt.Errorf("failed to switch to namespace %s with error: %+v", ns2, err)
	}
	peer, err = netlink.LinkByName(p1)
	if err != nil {
		return fmt.Errorf("failure to get pod's interface by name with error: %+v", err)
	}
	if err := netlink.LinkSetUp(peer); err != nil {
		return fmt.Errorf("failure setting link %s up with error: %+v", peer.Attrs().Name, err)
	}
	return nil
}

func setVethAddr(ns1, ns2 netns.NsHandle, p1, p2 string) error {

	var addr1 = &net.IPNet{IP: net.IPv4(1, 1, 1, 1), Mask: net.CIDRMask(30, 32)}
	var addr2 = &net.IPNet{IP: net.IPv4(1, 1, 1, 2), Mask: net.CIDRMask(30, 32)}
	var vethAddr1 = &netlink.Addr{IPNet: addr1, Peer: addr2}
	var vethAddr2 = &netlink.Addr{IPNet: addr2, Peer: addr1}

	ns, err := netns.Get()
	if err != nil {
		return fmt.Errorf("failed to get current process namespace")
	}
	defer netns.Set(ns)

	if err := netns.Set(ns1); err != nil {
		return fmt.Errorf("failed to switch to namespace %s with error: %+v", ns1, err)
	}
	link, err := netlink.LinkByName(p2)
	if err != nil {
		return fmt.Errorf("failure to get pod's interface by name with error: %+v", err)
	}
	if _, ok := link.(*netlink.Veth); !ok {
		return fmt.Errorf("failure, got unexpected interface type: %+v", reflect.TypeOf(link))
	}
	if err := netlink.AddrAdd(link, vethAddr1); err != nil {
		return fmt.Errorf("failure to assign IP to veth interface with error: %+v", err)
	}
	if err := netns.Set(ns2); err != nil {
		return fmt.Errorf("failed to switch to namespace %s with error: %+v", ns1, err)
	}
	peer, err := netlink.LinkByName(p1)
	if err != nil {
		return fmt.Errorf("failure to get pod's interface by name with error: %+v", err)
	}
	if _, ok := peer.(*netlink.Veth); !ok {
		return fmt.Errorf("failure, got unexpected interface type: %+v", reflect.TypeOf(link))
	}
	if err := netlink.AddrAdd(peer, vethAddr2); err != nil {
		return fmt.Errorf("failure to assign IP to veth interface with error: %+v", err)
	}
	return nil
}

func init() {
	runtime.LockOSThread()
}

func main() {
	var wg sync.WaitGroup

	flag.Parse()

	k8s, err := buildClient()
	if err != nil {
		logrus.Fatal("Failed to build kubernetes client, exitting...")
	}
	// Step 1: Start server on our dataplane socket, for now 2 API will be bound to this socket
	// DataplaneInterface API (eventually it will become the only API needed), currently used to notify
	// NSM for any changes in operational parameters and/or constraints and TestDataplane API which is used
	// to actually used to connect pods together.
	dataplaneController := DataplaneController{
		k8s: k8s,
		localMechanisms: []*common.LocalMechanism{
			&common.LocalMechanism{
				Type: common.LocalMechanismType_KERNEL_INTERFACE,
			},
		},
		updateCh: make(chan Update),
	}

	socket := *dataplane
	if err := tools.SocketCleanup(socket); err != nil {
		logrus.Fatalf("test-dataplane: failure to cleanup stale socket %s with error: %+v", socket, err)
	}
	dataplaneConn, err := net.Listen("unix", socket)
	if err != nil {
		logrus.Fatalf("test-dataplane: fail to open socket %s with error: %+v", socket, err)
	}
	dataplaneController.dataplaneServer = grpc.NewServer()
	// Binding testdataplane to gRPC server
	testdataplane.RegisterBuildConnectServer(dataplaneController.dataplaneServer, dataplaneController)
	testdataplane.RegisterDeleteConnectServer(dataplaneController.dataplaneServer, dataplaneController)
	// Binding dataplane Interface API to gRPC server
	dataplaneapi.RegisterDataplaneOperationsServer(dataplaneController.dataplaneServer, dataplaneController)

	go func() {
		wg.Add(1)
		if err := dataplaneController.dataplaneServer.Serve(dataplaneConn); err != nil {
			logrus.Fatalf("test-dataplane: failed to start grpc server on socket %s with error: %+v ", socket, err)
		}
	}()
	// Check if the socket of device plugin server is operation
	testSocket, err := tools.SocketOperationCheck(socket)
	if err != nil {
		logrus.Fatalf("test-dataplane: failure to communicate with the socket %s with error: %+v", socket, err)
	}
	testSocket.Close()
	logrus.Infof("test-dataplane: Test Dataplane controller is ready to serve...")
	for {
		// Step 2: The server is ready now dataplane needs to register with NSM.
		if _, err := os.Stat(registrarSocket); err != nil {
			logrus.Errorf("test-dataplane: failure to access nsm socket at %s with error: %+v, exiting...", registrarSocket, err)
			time.Sleep(time.Second * registrationRetryInterval)
			continue
		}

		conn, err := tools.SocketOperationCheck(registrarSocket)
		if err != nil {
			logrus.Errorf("test-dataplane: failure to communicate with the socket %s with error: %+v", registrarSocket, err)
			time.Sleep(time.Second * registrationRetryInterval)
			continue
		}
		defer conn.Close()
		logrus.Infof("DEBUG test-dataplane: connection to dataplane registrar socket % succeeded.", registrarSocket)

		registrarConnection := dataplaneregistrarapi.NewDataplaneRegistrationClient(conn)
		dataplane := dataplaneregistrarapi.DataplaneRegistrationRequest{
			DataplaneName:   "test-dataplane",
			DataplaneSocket: socket,
		}
		if _, err := registrarConnection.RequestDataplaneRegistration(context.Background(), &dataplane); err != nil {
			dataplaneController.dataplaneServer.Stop()
			logrus.Fatalf("test-dataplane: failure to communicate with the socket %s with error: %+v", registrarSocket, err)
			time.Sleep(time.Second * registrationRetryInterval)
			continue
		}
		logrus.Infof("DEBUG: test-dataplane: dataplane has successfully been registered, waiting for connection from NSM...")
		// Block on Liveness stream until NSM is gone, if failure of NSM is detected
		// go to a re-registration
		livenessMonitor(registrarConnection)
	}
}

func connectPods(k8s *kubernetes.Clientset, podName1, podName2, namespace1, namespace2 string) error {
	cid1, err := getContainerID(k8s, podName1, namespace1)
	if err != nil {
		return fmt.Errorf("Failed to get container ID for pod %s/%s with error: %+v", namespace1, podName1, err)
	}
	logrus.Printf("Discovered Container ID %s for pod %s/%s", cid1, namespace1, podName1)

	cid2, err := getContainerID(k8s, podName2, namespace2)
	if err != nil {
		return fmt.Errorf("Failed to get container ID for pod %s/%s with error: %+v", namespace2, podName2, err)
	}
	logrus.Printf("Discovered Container ID %s for pod %s/%s", cid2, namespace2, podName2)

	logrus.Printf("Debug: Calling getPidForContainer for container id %s", cid1)
	pid1, err := getPidForContainer(cid1)
	if err != nil {
		return fmt.Errorf("Failed to get Linux namespace for pod %s/%s with error: %+v", namespace1, podName1, err)
	}
	logrus.Printf("Debug: Calling getPidForContainer for container id %s", cid2)
	pid2, err := getPidForContainer(cid2)
	if err != nil {
		return fmt.Errorf("Failed to get Linux namespace for pod %s/%s with error: %+v", namespace2, podName2, err)
	}
	return connectProcesses(pid1, pid2)
}

func connectProcesses(pid1, pid2 int) error {
	ns1, err := netns.GetFromPid(pid1)
	if err != nil {
		return fmt.Errorf("Failed to get Linux namespace for pid %d with error: %+v", pid1, err)
	}
	ns2, err := netns.GetFromPid(pid2)
	if err != nil {
		return fmt.Errorf("Failed to get Linux namespace for pid %d with error: %+v", pid2, err)
	}

	intf1 := fmt.Sprintf("pid-%d", pid1)
	if len(intf1) > interfaceNameMaxLength {
		intf1 = intf1[:interfaceNameMaxLength]
	}
	intf2 := fmt.Sprintf("pid-%d", pid2)
	if len(intf2) > interfaceNameMaxLength {
		intf2 = intf2[:interfaceNameMaxLength]
	}

	if err := setVethPair(ns1, ns2, intf1, intf2); err != nil {
		return fmt.Errorf("Failed to get add veth pair to pids %d and %d with error: %+v",
			pid1, pid2, err)
	}

	if err := setVethAddr(ns1, ns2, intf1, intf2); err != nil {
		return fmt.Errorf("Failed to assign ip addresses to veth pair for pids %d and %d with error: %+v",
			pid1, pid2, err)
	}

	if err := listInterfaces(ns1); err != nil {
		logrus.Errorf("Failed to list interfaces of to %d with error: %+v", pid1, err)
	}
	if err := listInterfaces(ns2); err != nil {
		logrus.Errorf("Failed to list interfaces of %d with error: %+v", pid2, err)
	}
	return nil
}

func deleteLink(k8s *kubernetes.Clientset, podName string, namespace string, podType testdataplane.NSMPodType) error {
	cid, err := getContainerID(k8s, podName, namespace)
	if err != nil {
		return fmt.Errorf("Failed to get container ID for pod %s/%s with error: %+v", namespace, podName, err)
	}
	logrus.Printf("Discovered Container ID %s for pod %s/%s", cid, namespace, podName)

	ns, err := netns.GetFromDocker(cid)
	if err != nil {
		return fmt.Errorf("Failed to get Linux namespace for pod %s/%s with error: %+v", namespace, podName, err)
	}
	var interfacePrefix string
	switch podType {
	case testdataplane.NSMPodType_NSE:
		interfacePrefix = "nsm"

	case testdataplane.NSMPodType_NSMCLIENT:
		interfacePrefix = "nse"
	}
	if err := deleteVethInterface(ns, interfacePrefix); err != nil {
		return fmt.Errorf("Failed to veth interfacefor pod  %s/%s with error: %+v",
			namespace, podName, err)
	}
	return nil
}

func listInterfaces(targetNS netns.NsHandle) error {
	ns, err := netns.Get()
	if err != nil {
		return fmt.Errorf("failed to get current process namespace")
	}
	defer netns.Set(ns)

	if err := netns.Set(targetNS); err != nil {
		return fmt.Errorf("failed to switch to namespace %s with error: %+v", targetNS, err)
	}
	namespaceHandle, err := netlink.NewHandleAt(targetNS)
	if err != nil {
		return fmt.Errorf("failure to get pod's handle with error: %+v", err)
	}
	interfaces, err := namespaceHandle.LinkList()
	if err != nil {
		return fmt.Errorf("failure to get pod's interfaces with error: %+v", err)
	}
	logrus.Printf("pod's interfaces:")
	for _, intf := range interfaces {
		addrs, err := namespaceHandle.AddrList(intf, 0)
		if err != nil {
			logrus.Printf("failed to addresses for interface: %s with error: %v", intf.Attrs().Name, err)
		}
		logrus.Printf("Name: %s Type: %s Addresses: %+v", intf.Attrs().Name, intf.Type(), addrs)

	}
	return nil
}

func getPidForContainer(id string) (int, error) {
	pid := 0
	// memory is chosen randomly, any cgroup used by docker works
	cgroupType := "memory"

	cgroupRoot, err := findCgroupMountpoint(cgroupType)
	if err != nil {
		return pid, err
	}
	cgroupThis, err := getThisCgroup(cgroupType)
	if err != nil {
		return pid, err
	}

	id += "*"

	attempts := []string{
		filepath.Join(cgroupRoot, cgroupThis, id, "tasks"),
		// With more recent lxc versions use, cgroup will be in lxc/
		filepath.Join(cgroupRoot, cgroupThis, "lxc", id, "tasks"),
		// With more recent docker, cgroup will be in docker/
		filepath.Join(cgroupRoot, cgroupThis, "docker", id, "tasks"),
		// Even more recent docker versions under systemd use docker-<id>.scope/
		filepath.Join(cgroupRoot, "system.slice", "docker-"+id+".scope", "tasks"),
		// Even more recent docker versions under cgroup/systemd/docker/<id>/
		filepath.Join(cgroupRoot, "..", "systemd", "docker", id, "tasks"),
		// Kubernetes with docker and CNI is even more different
		filepath.Join(cgroupRoot, "..", "systemd", "kubepods", "*", "pod*", id, "tasks"),
		// Another flavor of containers location in recent kubernetes 1.11+
		filepath.Join(cgroupRoot, cgroupThis, "kubepods.slice", "kubepods-besteffort.slice", "*", "docker-"+id+".scope", "tasks"),
		// When runs inside of a container with recent kubernetes 1.11+
		filepath.Join(cgroupRoot, "kubepods.slice", "kubepods-besteffort.slice", "*", "docker-"+id+".scope", "tasks"),
		// For Docker in Docker container goes right under cgroupRoot
		filepath.Join(cgroupRoot, id+"*", "tasks"),
	}

	var filename string
	for _, attempt := range attempts {
		filenames, err := filepath.Glob(attempt)
		if err != nil {
			return pid, err
		}
		if filenames == nil {
			continue
		}
		if len(filenames) > 1 {
			return pid, fmt.Errorf("Ambiguous id supplied: %v", filenames)
		}
		if len(filenames) == 1 {
			filename = filenames[0]
			break
		}
	}

	if filename == "" {
		return pid, fmt.Errorf("Unable to find container: %v", id[:len(id)-1])
	}

	output, err := ioutil.ReadFile(filename)
	if err != nil {
		return pid, err
	}

	result := strings.Split(string(output), "\n")
	if len(result) == 0 || len(result[0]) == 0 {
		return pid, fmt.Errorf("No pid found for container")
	}

	pid, err = strconv.Atoi(result[0])
	if err != nil {
		return pid, fmt.Errorf("Invalid pid '%s': %s", result[0], err)
	}

	return pid, nil
}

// borrowed from docker/utils/utils.go
func findCgroupMountpoint(cgroupType string) (string, error) {
	output, err := ioutil.ReadFile("/proc/mounts")
	if err != nil {
		return "", err
	}

	// /proc/mounts has 6 fields per line, one mount per line, e.g.
	// cgroup /sys/fs/cgroup/devices cgroup rw,relatime,devices 0 0
	for _, line := range strings.Split(string(output), "\n") {
		parts := strings.Split(line, " ")
		if len(parts) == 6 && parts[2] == "cgroup" {
			for _, opt := range strings.Split(parts[3], ",") {
				if opt == cgroupType {
					return parts[1], nil
				}
			}
		}
	}

	return "", fmt.Errorf("cgroup mountpoint not found for %s", cgroupType)
}

// Returns the relative path to the cgroup docker is running in.
// borrowed from docker/utils/utils.go
// modified to get the docker pid instead of using /proc/self
func getThisCgroup(cgroupType string) (string, error) {
	dockerpid, err := ioutil.ReadFile("/var/run/docker.pid")
	if err != nil {
		return "", err
	}
	result := strings.Split(string(dockerpid), "\n")
	if len(result) == 0 || len(result[0]) == 0 {
		return "", fmt.Errorf("docker pid not found in /var/run/docker.pid")
	}
	pid, err := strconv.Atoi(result[0])
	if err != nil {
		return "", err
	}
	output, err := ioutil.ReadFile(fmt.Sprintf("/proc/%d/cgroup", pid))
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(output), "\n") {
		parts := strings.Split(line, ":")
		// any type used by docker should work
		if parts[1] == cgroupType {
			return parts[2], nil
		}
	}
	return "", fmt.Errorf("cgroup '%s' not found in /proc/%d/cgroup", cgroupType, pid)
}
