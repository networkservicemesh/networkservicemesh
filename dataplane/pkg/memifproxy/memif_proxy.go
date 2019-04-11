package memifproxy

import (
	"errors"
	"github.com/sirupsen/logrus"
	"net"
	"syscall"
)

const (
	bufferSize     = 128
	cmsgSize       = 24
	defaultNetwork = "unixpacket"
)

type Proxy struct {
	sourceSocket string
	targetSocket string
	network      string
	stopCh       chan struct{}
	errCh        chan error
}

type connectionResult struct {
	err  error
	conn *net.UnixConn
}

func NewProxy(sourceSocket, targetSocket string) *Proxy {
	return NewCustomProxy(sourceSocket, targetSocket, defaultNetwork)
}

func NewCustomProxy(sourceSocket, targetSocket, network string) *Proxy {
	return &Proxy{
		sourceSocket: sourceSocket,
		targetSocket: targetSocket,
		network:      network,
	}
}

func (mp *Proxy) Start() error {
	if mp.stopCh != nil {
		return errors.New("proxy is already started")
	}
	mp.stopCh = make(chan struct{}, 1)
	mp.errCh = make(chan error, 1)
	logrus.Infof("Request proxy source: %s, target: %s", mp.sourceSocket, mp.targetSocket)

	source, err := net.ResolveUnixAddr(mp.network, mp.sourceSocket)
	if err != nil {
		return err
	}
	logrus.Infof("Resolved source socket unix address: %v", mp.sourceSocket)

	target, err := net.ResolveUnixAddr(mp.network, mp.targetSocket)
	if err != nil {
		return err
	}
	logrus.Infof("Resolved target socket unix address: %v", mp.targetSocket)

	go func() {
		mp.errCh <- proxy(source, target, mp.network, mp.stopCh)
	}()
	return nil
}

func (mp *Proxy) Stop() error {
	if mp.stopCh == nil {
		return errors.New("proxy is not started")
	}
	close(mp.stopCh)
	err := <-mp.errCh
	close(mp.errCh)
	mp.stopCh = nil
	mp.errCh = nil
	return err
}

func proxy(source, target *net.UnixAddr, network string, stopCh <-chan struct{}) error {
	logrus.Info("Listening source socket...")
	sourceListener, err := net.ListenUnix(network, source)
	if err != nil {
		logrus.Error(err)
		return err
	}
	defer sourceListener.Close()
	sourceConn, err := acceptConnectionAsync(sourceListener, stopCh)

	if err != nil {
		logrus.Error(err)
		return err
	}

	if sourceConn == nil {
		return nil
	}

	defer sourceConn.Close()

	targetConn, err := connectToTargetAsync(target, network, stopCh)

	if err != nil {
		logrus.Error(err)
		return err
	}

	if targetConn == nil {
		return nil
	}

	defer targetConn.Close()

	sourceFd, closeSourceFd, err := getConnFd(sourceConn)
	if err != nil {
		logrus.Error(err)
		return err
	}
	defer closeSourceFd()
	logrus.Infof("Source connection fd: %v", sourceFd)

	targetFd, closeTargetFd, err := getConnFd(targetConn)
	if err != nil {
		logrus.Error(err)
		return err
	}
	defer closeTargetFd()
	logrus.Infof("Target connection fd: %v", targetFd)

	sourceStopCh := make(chan struct{})
	targetStopCh := make(chan struct{})

	go transfer(sourceFd, targetFd, sourceStopCh)
	go transfer(targetFd, sourceFd, targetStopCh)

	<-stopCh
	close(sourceStopCh)
	close(targetStopCh)
	logrus.Info("Proxy has stopped")
	return nil
}

func connectToTargetAsync(target *net.UnixAddr, network string, stopCh <-chan struct{}) (*net.UnixConn, error) {
	logrus.Info("Connecting to target socket...")
	connResCh := make(chan connectionResult, 1)
	go func() {
		defer close(connResCh)
		conn, err := net.DialUnix(network, nil, target)
		connResCh <- connectionResult{
			conn: conn,
			err:  err,
		}
		logrus.Info("Connected to target socket")
	}()
	for {
		select {
		case conn := <-connResCh:
			return conn.conn, conn.err
		case <-stopCh:
			logrus.Info("Connecting to target has stopped")
			return nil, nil
		}
	}
}

func acceptConnectionAsync(listener *net.UnixListener, stopCh <-chan struct{}) (*net.UnixConn, error) {
	logrus.Info("Accepting connections to source socket...")
	connResCh := make(chan connectionResult, 1)
	go func() {
		defer close(connResCh)
		conn, err := listener.AcceptUnix()
		connResCh <- connectionResult{
			conn: conn,
			err:  err,
		}
		logrus.Info("Connection from source socket successfully accepted")
	}()
	for {
		select {
		case conn := <-connResCh:
			return conn.conn, conn.err
		case <-stopCh:
			logrus.Info("Accept connection has stopped")
			return nil, nil
		}
	}
}

func transfer(fromFd, toFd int, stopCh <-chan struct{}) {
	dataBuffer := make([]byte, bufferSize)
	cmsgBuffer := make([]byte, cmsgSize)
	for {
		select {
		case <-stopCh:
			logrus.Infof("Transfer from %v to %v has benn stopped", fromFd, toFd)
			return
		default:
			dataN, cmsgN, _, _, err := syscall.Recvmsg(fromFd, dataBuffer, cmsgBuffer, 0)
			if err != nil {
				logrus.Error(err)
				return
			}
			logrus.Infof("Received message from %v", fromFd)
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
			logrus.Infof("Send message to %v", toFd)
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
