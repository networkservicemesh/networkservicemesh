package nsmvpp

import (
	"fmt"
	govppapi "git.fd.io/govpp.git/api"
	"github.com/docker/docker/pkg/mount"
	"github.com/ligato/networkservicemesh/dataplanes/vpp/bin_api/memif"
	"github.com/ligato/networkservicemesh/dataplanes/vpp/pkg/nsmutils"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/common"
	"github.com/sirupsen/logrus"
	"os"
	"path"
	"strconv"
)

const (
	BaseDir           = "/var/lib/networkservicemesh/"
	MemifDirectory    = "/memif"
	DefaultSocketFile = "memif.sock"
)

type createDirectory struct {
	path string
}

func (op *createDirectory) apply(apiCh govppapi.Channel) error {
	if err := os.MkdirAll(op.path, 0777); err != nil {
		return err
	}
	logrus.Infof("Create directory: %s", op.path)
	return nil
}

func (op *createDirectory) rollback() operation {
	return &deleteDirectory{
		path: op.path,
	}
}

type deleteDirectory struct {
	path string
}

func (op *deleteDirectory) apply(apiCh govppapi.Channel) error {
	if err := os.RemoveAll(op.path); err != nil {
		return err
	}
	logrus.Infof("Delete directory: %s", op.path)
	return nil
}

func (op *deleteDirectory) rollback() operation {
	return &createDirectory{
		path: op.path,
	}
}

type bindMount struct {
	device string
	target string
}

func (op *bindMount) apply(apiCh govppapi.Channel) error {
	if err := mount.Mount(op.device, op.target, "hard", "bind"); err != nil {
		return err
	}
	logrus.Infof("Successfully mount folder %s to %s", op.device, op.target)
	return nil
}

func (op *bindMount) rollback() operation {
	return &unbindMount{
		device: op.device,
		target: op.target,
	}
}

type unbindMount struct {
	device string
	target string
}

func (op *unbindMount) apply(apiCh govppapi.Channel) error {
	if err := mount.Unmount(op.target); err != nil {
		return err
	}
	logrus.Infof("Successfully unmount %s", op.target)
	return nil
}

func (op *unbindMount) rollback() operation {
	return &bindMount{
		device: op.device,
		target: op.target,
	}
}

type createSymlink struct {
	oldname string
	newname string
}

func (op *createSymlink) apply(apiCh govppapi.Channel) error {
	if err := os.Symlink(op.oldname, op.newname); err != nil {
		return fmt.Errorf("failed to create symlink: %s", err)
	}
	logrus.Info("Symlink successfully created")
	return nil
}

func (op *createSymlink) rollback() operation {
	return &deleteSymlink{
		oldname: op.oldname,
		newname: op.newname,
	}
}

type deleteSymlink struct {
	oldname string
	newname string
}

func (op *deleteSymlink) apply(apiCh govppapi.Channel) error {
	if err := os.RemoveAll(op.newname); err != nil {
		return err
	}
	logrus.Infof("Symlink successfully deleted")
	return nil
}

func (op *deleteSymlink) rollback() operation {
	return &createSymlink{
		oldname: op.oldname,
		newname: op.newname,
	}
}

type createMemifSocket struct {
	socketFilename string
	socketId       uint32
}

func (op *createMemifSocket) apply(apiCh govppapi.Channel) error {
	socketCreate := &memif.MemifSocketFilenameAddDel{
		IsAdd:          1,
		SocketID:       op.socketId,
		SocketFilename: []byte(op.socketFilename),
	}
	socketCreateReply := &memif.MemifSocketFilenameAddDelReply{}
	if err := apiCh.SendRequest(socketCreate).ReceiveReply(socketCreateReply); err != nil {
		return err
	}
	logrus.Infof("Memif socket %s successfully created", op.socketFilename)
	return nil
}

func (op *createMemifSocket) rollback() operation {
	return &deleteMemifSocket{
		socketFilename: op.socketFilename,
		socketId:       op.socketId,
	}
}

type deleteMemifSocket struct {
	socketFilename string
	socketId       uint32
}

func (op *deleteMemifSocket) apply(apiCh govppapi.Channel) error {
	socketDelete := &memif.MemifSocketFilenameAddDel{
		IsAdd:          0,
		SocketID:       op.socketId,
		SocketFilename: []byte(op.socketFilename),
	}
	socketDeleteReply := &memif.MemifSocketFilenameAddDelReply{}
	if err := apiCh.SendRequest(socketDelete).ReceiveReply(socketDeleteReply); err != nil {
		return err
	}
	logrus.Infof("Memif socket %s successfully deleted", op.socketFilename)
	return nil
}

func (op *deleteMemifSocket) rollback() operation {
	return &createMemifSocket{
		socketFilename: op.socketFilename,
		socketId:       op.socketId,
	}
}

type createMemifInterface struct {
	role     uint8
	socketId uint32
	intf     *vppInterface
}

func (op *createMemifInterface) apply(apiCh govppapi.Channel) error {
	interfaceCreate := &memif.MemifCreate{
		Role:     op.role,
		SocketID: op.socketId,
		ID:       29,
	}
	interfaceCreateReply := &memif.MemifCreateReply{}
	if err := apiCh.SendRequest(interfaceCreate).ReceiveReply(interfaceCreateReply); err != nil {
		return err
	}
	op.intf.id = interfaceCreateReply.SwIfIndex
	logrus.Infof("Interface successfully created: swIfIndex=%v", interfaceCreateReply.SwIfIndex)
	return nil
}

func (op *createMemifInterface) rollback() operation {
	return &deleteMemifInterface{
		role:     op.role,
		socketId: op.socketId,
		intf:     op.intf,
	}
}

type deleteMemifInterface struct {
	role     uint8
	socketId uint32
	intf     *vppInterface
}

func (op *deleteMemifInterface) apply(apiCh govppapi.Channel) error {
	interfaceDelete := &memif.MemifDelete{
		SwIfIndex: op.intf.id,
	}
	interfaceDeleteReply := &memif.MemifDeleteReply{}
	if err := apiCh.SendRequest(interfaceDelete).ReceiveReply(interfaceDeleteReply); err != nil {
		return err
	}
	logrus.Infof("Interface successfully deleted: swIfIndex=%v", op.intf.id)
	return nil
}

func (op *deleteMemifInterface) rollback() operation {
	return &createMemifInterface{
		role:     op.role,
		socketId: op.socketId,
		intf:     op.intf,
	}
}

func memifDirectConnect(src, dst map[string]string) ([]operation, error) {
	logrus.Info("Create memif-memif local connection requested")

	if err := validateMemif(src); err != nil {
		return nil, err
	}

	if err := validateMemif(dst); err != nil {
		return nil, err
	}

	negotiateParameters(src, dst)
	connectionId := buildConnectionId(src, dst)
	srcSocketDir := buildSocketDirPath(src, connectionId)
	dstSocketDir := buildSocketDirPath(dst, connectionId)

	operations := []operation{
		&createDirectory{path: srcSocketDir},
		&createDirectory{path: dstSocketDir},
		&bindMount{device: srcSocketDir, target: dstSocketDir},
	}

	if src[nsmutils.NSMSocketFile] != dst[nsmutils.NSMSocketFile] {
		master, slave := masterSlave(src, dst)
		operations = append(operations, &createSymlink{
			oldname: path.Join(srcSocketDir, master[nsmutils.NSMSocketFile]),
			newname: path.Join(srcSocketDir, slave[nsmutils.NSMSocketFile]),
		})
	}

	return operations, nil
}

func memifInterfaceCreate(c *createLocalInterface, apiCh govppapi.Channel) error {
	logrus.Infof("Creating memif interface with parameters: %v...", c.localMechanism.Parameters)

	if err := validateMemif(c.localMechanism.Parameters); err != nil {
		return err
	}

	var role uint8 = 0
	if getRole(c.localMechanism.Parameters) == nsmutils.NSMSlave {
		role = 1
	}
	socketFilename := path.Join(buildMemifDirPath(c.localMechanism.Parameters),
		c.localMechanism.Parameters[nsmutils.NSMSocketFile])

	operations := []operation{
		&createMemifSocket{
			socketFilename: socketFilename,
			socketId:       c.intf.socketId,
		},
		&createMemifInterface{
			role:     role,
			socketId: c.intf.socketId,
			intf:     c.intf,
		},
	}
	c.intf.mechanism = c.localMechanism

	pos, err := perform(operations, apiCh)

	if err != nil {
		rollback(operations, pos, apiCh)
		return err
	}

	return nil
}

func memifInterfaceDelete(op *deleteLocalInterface, apiCh govppapi.Channel) error {
	parameters := op.intf.mechanism.Parameters
	var socketId uint32 = 13
	var role uint8 = 0
	if getRole(parameters) == nsmutils.NSMMaster {
		role = 1
	}
	socketFilename := path.Join(buildMemifDirPath(parameters), parameters[nsmutils.NSMSocketFile])

	operations := []operation{
		&deleteMemifInterface{
			role:     role,
			socketId: socketId,
			intf:     op.intf,
		},
		&deleteMemifSocket{
			socketFilename: socketFilename,
			socketId:       socketId,
		},
	}

	pos, err := perform(operations, apiCh)

	if err != nil {
		rollback(operations, pos, apiCh)
		return err
	}

	return nil
}

func validateMemif(parameters map[string]string) error {
	keysList := nsmutils.Keys{
		nsmutils.NSMSocketFile:      nsmutils.KeyProperties{Validator: nsmutils.Any},
		nsmutils.NSMMaster:          nsmutils.KeyProperties{Validator: nsmutils.Bool},
		nsmutils.NSMSlave:           nsmutils.KeyProperties{Validator: nsmutils.Bool},
		nsmutils.NSMPerPodDirectory: nsmutils.KeyProperties{Mandatory: true, Validator: nsmutils.Any},
	}

	if err := nsmutils.ValidateParameters(parameters, keysList); err != nil {
		return err
	}

	_, hasMaster := parameters[nsmutils.NSMMaster]
	_, hasSlave := parameters[nsmutils.NSMSlave]

	if hasMaster && hasSlave {
		return fmt.Errorf("both master and slave parameter specified")
	}

	return nil
}

func buildConnectionId(src, dst map[string]string) string {
	return fmt.Sprintf("%d:%s:%d:%s",
		common.LocalMechanismType_MEM_INTERFACE,
		src[nsmutils.NSMPerPodDirectory],
		common.LocalMechanismType_MEM_INTERFACE,
		dst[nsmutils.NSMPerPodDirectory])
}

func negotiateParameters(src, dst map[string]string) error {
	negotiateSocketFile(src, dst)
	if err := negotiateRole(src, dst); err != nil {
		return err
	}
	return nil
}

func negotiateSocketFile(src, dst map[string]string) {
	if src[nsmutils.NSMSocketFile] == "" && dst[nsmutils.NSMSocketFile] == "" {
		logrus.Info("Socket files are not specified for both mechanisms")
		src[nsmutils.NSMSocketFile] = DefaultSocketFile
		dst[nsmutils.NSMSocketFile] = DefaultSocketFile
		return
	}

	if src[nsmutils.NSMSocketFile] == "" {
		src[nsmutils.NSMSocketFile] = dst[nsmutils.NSMSocketFile]
		return
	}

	if dst[nsmutils.NSMSocketFile] == "" {
		dst[nsmutils.NSMSocketFile] = src[nsmutils.NSMSocketFile]
		return
	}
}

func negotiateRole(src, dst map[string]string) error {
	if !hasRole(src) && !hasRole(dst) {
		src[nsmutils.NSMMaster] = strconv.FormatBool(true)
		dst[nsmutils.NSMSlave] = strconv.FormatBool(true)
		return nil
	}

	if !hasRole(src) {
		src[getOppositeRole(getRole(dst))] = strconv.FormatBool(true)
		return nil
	}

	if !hasRole(dst) {
		dst[getOppositeRole(getRole(src))] = strconv.FormatBool(true)
		return nil
	}

	if getRole(src) == getRole(dst) {
		return fmt.Errorf("mechanisms specified same roles")
	}

	return nil
}

func hasRole(parameters map[string]string) bool {
	_, b1 := parameters[nsmutils.NSMMaster]
	_, b2 := parameters[nsmutils.NSMSlave]

	return b1 || b2
}

func getOppositeRole(role string) string {
	if role == nsmutils.NSMMaster {
		return nsmutils.NSMSlave
	}
	return nsmutils.NSMMaster
}

func getRole(parameters map[string]string) string {
	if isMaster, _ := strconv.ParseBool(parameters[nsmutils.NSMMaster]); isMaster {
		return nsmutils.NSMMaster
	}
	return nsmutils.NSMSlave
}

func buildSocketDirPath(p map[string]string, name string) string {
	return path.Join(buildMemifDirPath(p), name)
}

func buildMemifDirPath(p map[string]string) string {
	return path.Join(BaseDir, p[nsmutils.NSMPerPodDirectory], MemifDirectory)
}

func masterSlave(src, dst map[string]string) (map[string]string, map[string]string) {
	if isMaster, _ := strconv.ParseBool(src[nsmutils.NSMMaster]); isMaster {
		return src, dst
	}
	return dst, src
}
