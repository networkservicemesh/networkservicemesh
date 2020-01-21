package sriovforwarder

import (
	"context"
	"runtime"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netns"
	"google.golang.org/grpc/status"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	mechanismMeta "github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/common"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/sriovkernel"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/sriovuserspace"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/forwarder/api/forwarder"
	"github.com/networkservicemesh/networkservicemesh/forwarder/pkg/common"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools/jaeger"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools/spanhelper"
	"github.com/networkservicemesh/networkservicemesh/utils/fs"
)

const (
	name = "sriov-forwarder"
)

// ConnectionSide is a helper type that specifies whether
// we deal with the destination or source interface
type ConnectionSide bool

const (
	src ConnectionSide = false
	dst ConnectionSide = true
)

// SRIOVForwarder represents SRIOV forwarder dataplane
type SRIOVForwarder struct {
	common *common.ForwarderConfig
	// TODO: monitoring *monitoring.Metrics
}

// VFInterfaceConfiguration represents configuration details that
// will be used to setup or close cross connection
type VFInterfaceConfiguration struct {
	pciAddress  string
	name        string
	ipAddress   string
	macAddress  string
	targetNetns string
}

// CreateSRIOVForwarder returns new instance of SRIOV forwarder
func CreateSRIOVForwarder() *SRIOVForwarder {
	return &SRIOVForwarder{}
}

// Init initializes SRIOV forwarder dataplane configuration
func (fwd *SRIOVForwarder) Init(commonConfig *common.ForwarderConfig) error {
	logrus.Infof("Initializing %s...", name)

	fwd.common = commonConfig
	fwd.common.Name = name

	closer := jaeger.InitJaeger(fwd.common.Name)
	defer func() { _ = closer.Close() }()

	fwd.common.MechanismsUpdateChannel = make(chan *common.Mechanisms, 1)
	fwd.common.Mechanisms = &common.Mechanisms{
		LocalMechanisms: []*connection.Mechanism{
			{Type: sriovkernel.MECHANISM},
			{Type: sriovuserspace.MECHANISM},
		},
		RemoteMechanisms: []*connection.Mechanism{},
	}

	// TODO: metrics (start monitoring)

	return nil
}

// CreateForwarderServer returns instance of SRIOV forwarder dataplane server
func (fwd *SRIOVForwarder) CreateForwarderServer(config *common.ForwarderConfig) forwarder.ForwarderServer {
	return fwd
}

// Close closes connection
func (fwd *SRIOVForwarder) Close(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*empty.Empty, error) {
	logrus.Infof("Close() called with %+v", crossConnect)

	span := spanhelper.FromContext(ctx, "Close")
	defer span.Finish()

	var err error

	// check whether the forwarding plane supports the requested connection type
	if err := common.SanityCheckConnectionType(fwd.common.Mechanisms, crossConnect); err != nil {
		logrus.Errorf("mechanism type not supported: %v", err)
		return &empty.Empty{}, err
	}

	mechanismType := crossConnect.GetLocalSource().GetMechanism().GetType()
	switch mechanismType {
	case sriovkernel.MECHANISM:
		err = fwd.closeLocalKernelInterfaceConnection(crossConnect)
	case sriovuserspace.MECHANISM:
		err = fwd.closeLocalUserspaceInterfaceConnection(crossConnect)
	// TODO: add remote mechanisms
	default:
		err = errors.Errorf("mechanism type %q not supported", mechanismType)
	}

	return &empty.Empty{}, err
}

// Request performs connection setup
func (fwd *SRIOVForwarder) Request(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*crossconnect.CrossConnect, error) {
	logrus.Infof("Request() called with %+v", crossConnect)
	var err error

	span := spanhelper.FromContext(ctx, "Request")
	defer span.Finish()

	// TODO: metrics

	// check whether the forwarding plane supports the requested connection type
	if err = common.SanityCheckConnectionType(fwd.common.Mechanisms, crossConnect); err != nil {
		logrus.Errorf("%s: mechanism type not supported: %v", fwd.common.Name, err)
		return crossConnect, err
	}

	mechanismType := crossConnect.GetLocalSource().GetMechanism().GetType()
	switch mechanismType {
	case sriovkernel.MECHANISM:
		err = fwd.setupLocalKernelInterfaceConnection(crossConnect)
	case sriovuserspace.MECHANISM:
		err = fwd.setupLocalUserspaceConnection(crossConnect)
	// TODO: add remote mechanisms
	default:
		err = errors.Errorf("mechanism type %q not supported", mechanismType)
	}

	// TODO: monitoring update if no errors

	return crossConnect, err
}

func getLocalConnectionConfig(c *connection.Connection, side ConnectionSide) VFInterfaceConfiguration {
	name, ok := c.GetMechanism().GetParameters()[mechanismMeta.InterfaceNameKey]
	if !ok {
		name = c.GetMechanism().GetParameters()[mechanismMeta.Workspace]
	}

	var ipAddress string
	if side == dst {
		ipAddress = c.GetContext().GetIpContext().GetDstIpAddr()
	} else {
		ipAddress = c.GetContext().GetIpContext().GetSrcIpAddr()
	}

	return VFInterfaceConfiguration{
		pciAddress:  c.GetMechanism().GetParameters()[sriovkernel.PCIAddress], // this key name is common for userspace and kernel
		targetNetns: c.GetMechanism().GetParameters()[mechanismMeta.NetNsInodeKey],
		name:        name,
		ipAddress:   ipAddress,
	}
}

func (fwd *SRIOVForwarder) setupLocalKernelInterfaceConnection(cc *crossconnect.CrossConnect) error {
	logrus.Infof("setting up local kernel interface connection")

	// lock the OS thread so we don't accidentally switch namespaces
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// TODO: make use of these guys
	//srcRoutes := cc.GetLocalSource().GetContext().GetIpContext().GetDstRoutes()
	//neighbors := cc.GetLocalSource().GetContext().GetIpContext().GetIpNeighbors()
	//dstRoutes := cc.GetLocalDestination().GetContext().GetIpContext().GetSrcRoutes()

	logrus.Infof("local source request: %s", cc.GetLocalSource())

	// get source VF interface configuration - TODO: validate these parameters
	sourceConfig := getLocalConnectionConfig(cc.GetLocalSource(), src)

	// configure source VF interface
	err := setupVF(sourceConfig)
	if err != nil {
		logrus.Errorf("error configuring local source VF: %s", err)
		return err
	}

	logrus.Infof("local destination request: %s", cc.GetLocalDestination())

	// get destination VF interface configuration - TODO: validate these parameters
	destinationConfig := getLocalConnectionConfig(cc.GetLocalDestination(), dst)

	// configure destination VF interface
	err = setupVF(destinationConfig)
	if err != nil {
		logrus.Errorf("error configuring local destination VF: %s", err)
		return err
	}

	return nil
}

func (fwd *SRIOVForwarder) closeLocalKernelInterfaceConnection(cc *crossconnect.CrossConnect) error {
	logrus.Infof("setting up local kernel interface connection")

	// lock the OS thread so we don't accidentally switch namespaces
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	logrus.Infof("local source close request: %s", cc.GetLocalSource())

	// get source VF interface configuration - TODO: validate these parameters
	sourceConfig := getLocalConnectionConfig(cc.GetLocalSource(), src)

	// release source VF interface
	err := releaseVF(sourceConfig)
	if err != nil {
		logrus.Errorf("error releasing local source VF: %s", err)
		return err
	}

	logrus.Infof("local destination close request: %s", cc.GetLocalDestination())

	// get destination VF interface configuration - TODO: validate these parameters
	destinationConfig := getLocalConnectionConfig(cc.GetLocalDestination(), dst)

	// release destination VF interface
	err = releaseVF(destinationConfig)
	if err != nil {
		logrus.Errorf("error releasing local destination VF: %s", err)
		return err
	}

	return nil
}

func (fwd *SRIOVForwarder) setupLocalUserspaceConnection(cc *crossconnect.CrossConnect) error {
	logrus.Infof("setting up local userspace connection")

	// TODO: decide what to do:
	// - put a file with metadata in pod volume?
	// - annotate the pod and then use downwardAPI in the pod spec?
	// - something else?
	// - only return some metadata to the requesting side and let it do the stuff

	return nil
}

func (fwd *SRIOVForwarder) closeLocalUserspaceInterfaceConnection(cc *crossconnect.CrossConnect) error {
	return nil
}

// MonitorMechanisms sends monitoring mechanisms update to update server
func (fwd *SRIOVForwarder) MonitorMechanisms(empty *empty.Empty, updateSrv forwarder.MechanismsMonitor_MonitorMechanismsServer) error {
	initialUpdate := &forwarder.MechanismUpdate{
		LocalMechanisms:  fwd.common.Mechanisms.LocalMechanisms,
		RemoteMechanisms: fwd.common.Mechanisms.RemoteMechanisms,
	}

	logrus.Infof("%s: sending MonitorMechanisms update: %v", fwd.common.Name, initialUpdate)
	if err := updateSrv.Send(initialUpdate); err != nil {
		logrus.Errorf("%s: detected server error %s, gRPC code: %+v on gRPC channel", fwd.common.Name, err.Error(), status.Convert(err).Code())
		return nil
	}

	for update := range fwd.common.MechanismsUpdateChannel {
		fwd.common.Mechanisms = update
		logrus.Infof("%s: sending MonitorMechanisms update: %v", fwd.common.Name, update)
		if err := updateSrv.Send(&forwarder.MechanismUpdate{
			RemoteMechanisms: update.RemoteMechanisms,
			LocalMechanisms:  update.LocalMechanisms,
		}); err != nil {
			logrus.Errorf("%s: detected server error %s, gRPC code: %+v on gRPC channel", fwd.common.Name, err.Error(), status.Convert(err).Code())
			return nil
		}
	}
	return nil
}

func setupVF(config VFInterfaceConfiguration) error {
	// host network namespace to switch back to after finishing link setup
	hostNetns, err := netns.Get()
	if err != nil {
		return errors.Errorf("failed to get host namespace: %v", err)
	}
	defer hostNetns.Close()

	// always switch back to the host namespace at the end of link setup
	defer func() {
		if err = netns.Set(hostNetns); err != nil {
			logrus.Errorf("failed to switch back to host namespace: %v", err)
		}
	}()

	// get network namespace handle
	targetNetns, err := fs.GetNsHandleFromInode(config.targetNetns)
	if err != nil {
		return errors.Wrap(err, "failed to setup VF")
	}
	defer targetNetns.Close()

	// get VF link representor
	link, err := GetLink(config.pciAddress, config.name, hostNetns, targetNetns)
	if err != nil {
		return errors.Wrap(err, "failed to setup VF")
	}

	// move link into pod's network namespace
	err = link.MoveToNetns(targetNetns)
	if err != nil {
		return errors.Wrap(err, "failed to setup VF")
	}

	// switch to pod's network namespace to apply configuration, link is already there
	err = netns.Set(targetNetns)
	if err != nil {
		return errors.Wrap(err, "failed to setup VF")
	}

	// add IP address
	err = link.AddAddress(config.ipAddress)
	if err != nil {
		return errors.Wrap(err, "failed to setup VF")
	}

	// set new interface name
	err = link.SetName(config.name)
	if err != nil {
		return err
	}

	// bring up the link
	err = link.SetAdminState(UP)
	if err != nil {
		return err
	}

	// TODO: set MAC address, routes, neighbours, vlan and other properties etc.

	return nil
}

func releaseVF(config VFInterfaceConfiguration) error {
	// host network namespace to switch back to after finishing link setup
	hostNetns, err := netns.Get()
	if err != nil {
		return errors.Errorf("failed to get host namespace: %v", err)
	}
	defer hostNetns.Close()

	// always switch back to the host namespace at the end of link setup
	defer func() {
		if err = netns.Set(hostNetns); err != nil {
			logrus.Errorf("failed to switch back to host namespace: %v", err)
		}
	}()

	// get network namespace handle
	targetNetns, err := fs.GetNsHandleFromInode(config.targetNetns)
	if err != nil {
		return errors.Wrap(err, "failed to release VF")
	}
	defer targetNetns.Close()

	// get VF link representor
	link, err := GetLink(config.pciAddress, config.name, targetNetns)
	if err != nil {
		return errors.Wrap(err, "failed to release VF")
	}

	// switch to pod's network namespace to apply configuration, link is already there
	err = netns.Set(targetNetns)
	if err != nil {
		return errors.Wrap(err, "failed to release VF")
	}

	// delete IP address
	err = link.DeleteAddress(config.ipAddress)
	if err != nil {
		return errors.Wrapf(err, "failed to release VF")
	}

	// TODO: move VF back to the host ns if no other connections in workspace

	// TODO: switch to host namespace

	// TODO(optional): restore original link name

	return nil
}
