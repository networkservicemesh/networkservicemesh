package memifproxy

import (
	"net"
	"os"
	"syscall"

	"github.com/pkg/errors"

	"github.com/sirupsen/logrus"
)

const (
	bufferSize     = 128
	cmsgSize       = 24
	defaultNetwork = "unixpacket"
)

// StopListenerAdapter adapts func() to Listener interface
type StopListenerAdapter func()

// OnStopped occurs when proxy stopped
func (f StopListenerAdapter) OnStopped() {
	if f != nil {
		f()
	}
}

// Listener is a custom event handler
type Listener interface {
	OnStopped()
}

// Proxy is a proxy for direct MEMIF connections
type Proxy interface {
	Stop() error
	Start() error
}

type proxyImpl struct {
	listener       Listener
	network        string
	stopCh         chan struct{}
	errCh          chan error
	sourceListener *net.UnixListener
	source         *net.UnixAddr
	target         *net.UnixAddr
}

type connectionResult struct {
	err  error
	conn *net.UnixConn
}

//NewWithListener creates a new proxy for memif connection with specific proxy event listener
func NewWithListener(sourceSocket, targetSocket string, listener Listener) (Proxy, error) {
	return newCustomProxy(sourceSocket, targetSocket, defaultNetwork, listener)
}

//New creates a new proxy for memif connection
func New(sourceSocket, targetSocket string) (Proxy, error) {
	return NewWithListener(sourceSocket, targetSocket, nil)
}

func newCustomProxy(sourceSocket, targetSocket, network string, listener Listener) (Proxy, error) {
	source, err := net.ResolveUnixAddr(network, sourceSocket)
	if err != nil {
		return nil, err
	}
	logrus.Infof("Resolved source socket unix address: %v", source)

	target, err := net.ResolveUnixAddr(network, targetSocket)
	if err != nil {
		return nil, err
	}

	logrus.Infof("Resolved target socket unix address: %v", target)
	if err := tryDeleteFileIfExist(sourceSocket); err != nil {
		logrus.Errorf("An error during source socket file deleting %v", err.Error())
		return nil, err
	}
	return &proxyImpl{
		source:   source,
		target:   target,
		network:  network,
		listener: listener,
	}, nil
}

//Start means  start listen to source socket and wait for new connections in a separate goroutine
func (p *proxyImpl) Start() error {
	if p.sourceListener != nil {
		return errors.New("proxy is already started")
	}
	var err error
	p.sourceListener, err = net.ListenUnix(p.network, p.source)
	if err != nil {
		logrus.Errorf("can't listen unix %v", err)
		return err
	}
	logrus.Info("Listening source socket...")

	p.stopCh = make(chan struct{}, 1)
	p.errCh = make(chan error, 1)

	go func() {
		p.errCh <- p.proxy()
		if p.listener != nil {
			p.listener.OnStopped()
		}
	}()
	return nil
}

//Stop means stop listen to source socket and close  connections
func (p *proxyImpl) Stop() error {
	if p.sourceListener == nil {
		return errors.New("proxy is not started")
	}
	close(p.stopCh)
	err := <-p.errCh
	close(p.errCh)
	if err != nil {
		logrus.Error(err)
	}
	err = p.sourceListener.Close()
	if err != nil {
		logrus.Error(err)
	}
	p.sourceListener = nil
	return err
}

func (p *proxyImpl) proxy() error {
	sourceConn, err := acceptConnectionAsync(p.sourceListener, p.stopCh)
	if err != nil {
		return err
	}
	if sourceConn == nil {
		return nil
	}
	defer sourceConn.Close()

	targetConn, err := connectToTargetAsync(p.target, p.network, p.stopCh)
	if err != nil {
		return err
	}
	if targetConn == nil {
		return nil
	}

	defer targetConn.Close()

	sourceFd, closeSourceFd, err := getConnFd(sourceConn)
	if err != nil {
		logrus.Errorf("can't get source conn fd %v", err)
		return err
	}
	defer closeSourceFd()
	logrus.Infof("Source connection fd: %v", sourceFd)

	targetFd, closeTargetFd, err := getConnFd(targetConn)
	if err != nil {
		logrus.Errorf("can't get target conn fd %v", err)
		return err
	}
	defer closeTargetFd()
	logrus.Infof("Target connection fd: %v", targetFd)

	sourceStopCh := make(chan struct{})
	targetStopCh := make(chan struct{})

	go transfer(sourceFd, targetFd, sourceStopCh)
	go transfer(targetFd, sourceFd, targetStopCh)

	select {
	case <-p.stopCh:
		break
	case <-sourceStopCh:
		break
	case <-targetStopCh:
		break
	}
	logrus.Info("Proxy has been stopped")
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

func transfer(fromFd, toFd int, stopCh chan struct{}) {
	dataBuffer := make([]byte, bufferSize)
	cmsgBuffer := make([]byte, cmsgSize)
	defer close(stopCh)
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

func tryDeleteFileIfExist(path string) error {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err == nil {
		return os.Remove(path)
	}
	return err
}
