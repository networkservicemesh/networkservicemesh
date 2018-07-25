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
	"log"
	"net"
	"os"
	"reflect"
	"runtime"
	"strings"
	"sync"

	"github.com/ligato/networkservicemesh/pkg/nsm/apis/simpledataplane"
	"github.com/ligato/networkservicemesh/pkg/tools"
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
	dataplaneSocket        = "/var/lib/networkservicemesh/dataplane.sock"
	interfaceNameMaxLength = 15
)

var (
	dataplane  = flag.String("dataplane-socket", dataplaneSocket, "Location of the dataplane gRPC socket")
	kubeconfig = flag.String("kubeconfig", "", "Absolute path to the kubeconfig file. Either this or master needs to be set if the provisioner is being run out of cluster.")
)

// DataplaneController keeps k8s client and gRPC server
type DataplaneController struct {
	k8s             *kubernetes.Clientset
	dataplaneServer *grpc.Server
}

// RequestDeleteConnect implements method for simpledataplane proto
func (d DataplaneController) RequestDeleteConnect(ctx context.Context, in *simpledataplane.DeleteConnectRequest) (*simpledataplane.DeleteConnectReply, error) {

	podName := in.Pod.Metadata.Name
	if podName == "" {
		logrus.Error("simple-dataplane: missing required pod name")
		return &simpledataplane.DeleteConnectReply{
			Deleted:     false,
			DeleteError: fmt.Sprint("missing required name for pod"),
		}, status.Error(codes.NotFound, "missing required pod name")
	}
	podNamespace := "default"
	if in.Pod.Metadata.Namespace != "" {
		podNamespace = in.Pod.Metadata.Namespace
	}

	switch in.PodType {
	case simpledataplane.NSMPodType_NSE:
	case simpledataplane.NSMPodType_NSMCLIENT:
	default:
		logrus.Error("simple-dataplane: invalid pod type")
		return &simpledataplane.DeleteConnectReply{
			Deleted:     false,
			DeleteError: fmt.Sprintf("invalid pod type detected for pod %s/%s", podNamespace, podName),
		}, status.Error(codes.Aborted, "nvalid pod type")
	}
	if err := deleteLink(d.k8s, podName, podNamespace, in.PodType); err != nil {
		return &simpledataplane.DeleteConnectReply{
			Deleted:     false,
			DeleteError: fmt.Sprintf("failed to delete interface(s) for pod %s/%s with error: %+v", podNamespace, podName, err),
		}, status.Error(codes.Aborted, "failed to delete interface(s) for pod")
	}

	return &simpledataplane.DeleteConnectReply{
		Deleted: true,
	}, nil
}

// RequestBuildConnect implements method for simpledataplane proto
func (d DataplaneController) RequestBuildConnect(ctx context.Context, in *simpledataplane.BuildConnectRequest) (*simpledataplane.BuildConnectReply, error) {

	podName1 := in.SourcePod.Metadata.Name
	if podName1 == "" {
		logrus.Error("simple-dataplane: missing required pod name")
		return &simpledataplane.BuildConnectReply{
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
		logrus.Error("simple-dataplane: missing required pod name")
		return &simpledataplane.BuildConnectReply{
			Built:      false,
			BuildError: fmt.Sprint("missing required name for pod 2"),
		}, status.Error(codes.NotFound, "missing required pod name")
	}
	podNamespace2 := "default"
	if in.DestinationPod.Metadata.Namespace != "" {
		podNamespace2 = in.DestinationPod.Metadata.Namespace
	}

	logrus.Infof("simple-dataplane: attempting to interconnect pods %s/%s and %s/%s",
		podNamespace1,
		podName1,
		podNamespace2,
		podName2)
	// TODO (sbezverk) Add ip address check
	if err := connectPods(d.k8s, podName1, podName2, podNamespace1, podNamespace2); err != nil {
		logrus.Errorf("simple-dataplane: failed to interconnect pods %s/%s and %s/%s with error: %+v",
			podNamespace1,
			podName1,
			podNamespace2,
			podName2,
			err)
		return &simpledataplane.BuildConnectReply{
			Built: false,
			BuildError: fmt.Sprintf("simple-dataplane: failed to interconnect pods %s/%s and %s/%s with error: %+v",
				podNamespace1,
				podName1,
				podNamespace2,
				podName2,
				err),
		}, status.Error(codes.Aborted, "failure to interconnect pods")
	}

	return &simpledataplane.BuildConnectReply{
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
			return "", fmt.Errorf("simple-dataplane: none of containers of pod %s/%s is in running state", p.ObjectMeta.Namespace, p.ObjectMeta.Name)
		}
	}

	return "", fmt.Errorf("simple-dataplane: pod %s/%s not found", ns, pn)
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
	if err := namespaceHandle.LinkAdd(veth); err != nil {
		return fmt.Errorf("failure to add veth to pod with error: %+v", err)
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

	dataplaneController := DataplaneController{k8s: k8s}

	socket := *dataplane
	if err := tools.SocketCleanup(socket); err != nil {
		logrus.Fatalf("simple-dataplane: failure to cleanup stale socket %s with error: %+v", socket, err)
	}
	dataplaneConn, err := net.Listen("unix", socket)
	if err != nil {
		logrus.Fatalf("simple-dataplane: fail to open socket %s with error: %+v", socket, err)
	}
	dataplaneController.dataplaneServer = grpc.NewServer()
	simpledataplane.RegisterBuildConnectServer(dataplaneController.dataplaneServer, dataplaneController)

	go func() {
		wg.Add(1)
		if err := dataplaneController.dataplaneServer.Serve(dataplaneConn); err != nil {
			logrus.Fatalf("simple-dataplane: failed to start grpc server on socket %s with error: %+v ", socket, err)
		}
	}()
	// Check if the socket of device plugin server is operation
	testSocket, err := tools.SocketOperationCheck(socket)
	if err != nil {
		logrus.Fatalf("simple-dataplane: failure to communicate with the socket %s with error: %+v", socket, err)
	}
	testSocket.Close()

	logrus.Infof("simple-dataplane: Simple Dataplane controller is ready to serve...")
	// Now block on WaitGroup
	wg.Wait()
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

	ns1, err := netns.GetFromDocker(cid1)
	if err != nil {
		return fmt.Errorf("Failed to get Linux namespace for pod %s/%s with error: %+v", namespace1, podName1, err)
	}
	ns2, err := netns.GetFromDocker(cid2)
	if err != nil {
		return fmt.Errorf("Failed to get Linux namespace for pod %s/%s with error: %+v", namespace2, podName2, err)
	}

	podName1 = "nse-" + podName1[:interfaceNameMaxLength-4]
	podName2 = "nsm-" + podName2[:interfaceNameMaxLength-4]

	if err := setVethPair(ns1, ns2, podName1, podName2); err != nil {
		return fmt.Errorf("Failed to get add veth pair to pods %s/%s and %s/%s with error: %+v",
			namespace1, podName1, namespace2, podName2, err)
	}

	if err := setVethAddr(ns1, ns2, podName1, podName2); err != nil {
		return fmt.Errorf("Failed to assign ip addresses to veth pair for pods %s/%s and %s/%s with error: %+v",
			namespace1, podName1, namespace2, podName2, err)
	}

	if err := listInterfaces(ns1); err != nil {
		logrus.Errorf("Failed to list interfaces of to %s/%swith error: %+v", namespace1, podName1, err)
	}
	if err := listInterfaces(ns2); err != nil {
		logrus.Errorf("Failed to list interfaces of %s/%swith error: %+v", namespace2, podName2, err)
	}
	return nil
}

func deleteLink(k8s *kubernetes.Clientset, podName string, namespace string, podType simpledataplane.NSMPodType) error {
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
	case simpledataplane.NSMPodType_NSE:
		interfacePrefix = "nsm"

	case simpledataplane.NSMPodType_NSMCLIENT:
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
	log.Printf("pod's interfaces:")
	for _, intf := range interfaces {
		addrs, err := namespaceHandle.AddrList(intf, 0)
		if err != nil {
			log.Printf("failed to addresses for interface: %s with error: %v", intf.Attrs().Name, err)
		}
		log.Printf("Name: %s Type: %s Addresses: %+v", intf.Attrs().Name, intf.Type(), addrs)

	}
	return nil
}
