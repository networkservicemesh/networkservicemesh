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

	"github.com/vishvananda/netns"

	"github.com/vishvananda/netlink"

	"github.com/ligato/networkservicemesh/pkg/nsm/apis/simpledataplane"
	"github.com/ligato/networkservicemesh/pkg/tools"
	"github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	dataplaneSocket = "/var/lib/networkservicemesh/dataplane.sock"
)

var (
	dataplane  = flag.String("dataplane-scoket", dataplaneSocket, "Location of the dataplane gRPC socket")
	kubeconfig = flag.String("kubeconfig", "", "Absolute path to the kubeconfig file. Either this or master needs to be set if the provisioner is being run out of cluster.")
)

// DataplaneController keeps k8s client and gRPC server
type DataplaneController struct {
	k8s             *kubernetes.Clientset
	dataplaneServer *grpc.Server
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

	if err := connectPods(d.k8s, podName1, podName2, podNamespace1, podNamespace2); err != nil {
		logrus.Error("simple-dataplane: failed to interconnect pods %s/%s and %s/%s with error: %+v",
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
			return strings.Split(p.Status.ContainerStatuses[0].ContainerID, "://")[1][:12], nil
		}
	}

	return "", nil
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
	peer, err := netlink.LinkByName(p1)
	if err != nil {
		return fmt.Errorf("failure to get pod's interface by name with error: %+v", err)
	}
	if _, ok := peer.(*netlink.Veth); !ok {
		return fmt.Errorf("failure, got unexpected interface type: %+v", reflect.TypeOf(link))
	}
	if err := netlink.LinkSetNsFd(peer, int(ns2)); err != nil {
		return fmt.Errorf("failure to get place veth into peer's pod with error: %+v", err)
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
		logrus.Fatalf("nse: failure to communicate with the socket %s with error: %+v", socket, err)
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
