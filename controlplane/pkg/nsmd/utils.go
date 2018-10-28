package nsmd

import "net"

type customListener struct {
	net.Listener
	serverSocket string
}

type customConn struct {
	net.Conn
	localAddr *net.UnixAddr
}

func (c customConn) RemoteAddr() net.Addr {
	return c.localAddr
}

func newCustomListener(socket string) (customListener, error) {
	listener, err := net.Listen("unix", socket)
	if err == nil {
		custList := customListener{
			Listener:     listener,
			serverSocket: socket,
		}
		return custList, nil
	}
	return customListener{}, err
}

func (l customListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return &customConn{
		Conn:      conn,
		localAddr: &net.UnixAddr{Net: "unix", Name: l.serverSocket},
	}, nil
}
