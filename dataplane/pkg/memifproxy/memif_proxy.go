package memifproxy

import (
	"github.com/sirupsen/logrus"
	"net"
	"sync"
	"syscall"
)

const (
	bufferSize = 128
	cmsgSize   = 24
	network    = "unixpacket"
)

type Proxy struct {
	sourceSocket string
	targetSocket string
	stopCh       chan struct{}
}

func NewProxy(sourceSocket, targetSocket string) *Proxy {
	return &Proxy{
		sourceSocket: sourceSocket,
		targetSocket: targetSocket,
	}
}

func (mp *Proxy) Start() error {
	logrus.Infof("Request proxy source: %s, target: %s", mp.sourceSocket, mp.targetSocket)

	mp.stopCh = make(chan struct{})

	logrus.Infof("Resolving source socket unix address: %v", mp.sourceSocket)
	source, err := net.ResolveUnixAddr(network, mp.sourceSocket)
	if err != nil {
		return err
	}

	logrus.Infof("Resolving target socket unix address: %v", mp.targetSocket)
	target, err := net.ResolveUnixAddr(network, mp.targetSocket)
	if err != nil {
		return err
	}

	go proxy(source, target, mp.stopCh)
	return nil
}

func (mp *Proxy) Stop() {
	close(mp.stopCh)
}

func proxy(source, target *net.UnixAddr, stopCh <-chan struct{}) {
	logrus.Info("Listening source socket...")
	sourceListener, err := net.ListenUnix(network, source)
	if err != nil {
		logrus.Error(err)
		return
	}
	logrus.Info("Accepting connections to source socket...")
	sourceConn, err := sourceListener.AcceptUnix()
	if err != nil {
		logrus.Error(err)
		return
	}
	logrus.Info("Connection from source socket successfully accepted")

	logrus.Info("Connecting to target socket...")
	targetConn, err := net.DialUnix(network, nil, target)
	if err != nil {
		logrus.Error(err)
		return
	}
	logrus.Info("Successfully connected to target socket")

	sourceFd, closeSourceFd, err := getConnFd(sourceConn)
	if err != nil {
		logrus.Error(err)
		return
	}
	defer closeSourceFd()
	logrus.Infof("Source connection fd: %v", sourceFd)

	targetFd, closeTargetFd, err := getConnFd(targetConn)
	if err != nil {
		logrus.Error(err)
		return
	}
	defer closeTargetFd()
	logrus.Infof("Target connection fd: %v", targetFd)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		transfer(sourceFd, targetFd, stopCh)
	}()

	go func() {
		defer wg.Done()
		transfer(targetFd, sourceFd, stopCh)
	}()

	wg.Wait()
}

func transfer(fromFd, toFd int, stopCh <-chan struct{}) {
	dataBuffer := make([]byte, bufferSize)
	cmsgBuffer := make([]byte, cmsgSize)
	for {
		select {
		case <-stopCh:
			break
		default:
			dataN, cmsgN, _, _, err := syscall.Recvmsg(fromFd, dataBuffer, cmsgBuffer, 0)
			if err != nil {
				logrus.Error(err)
				return
			}
			var sendDataBuf []byte = nil
			if dataN > 0 {
				sendDataBuf = dataBuffer
			}
			var sendCmsgBuf []byte = nil
			if cmsgN > 0 {
				sendCmsgBuf = cmsgBuffer
			}
			if err := syscall.Sendmsg(toFd, sendDataBuf, sendCmsgBuf, nil, 0); err != nil {
				logrus.Error(err)
				return
			}
		}
	}
}

func getConnFd(conn *net.UnixConn) (int, func(), error) {
	file, err := conn.File()
	if err != nil {
		return -1, func() {}, err
	}

	fd := int(file.Fd())
	return fd, func() { file.Close() }, nil
}
