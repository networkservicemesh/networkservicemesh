package proxynsmd

import (
	"net"
)

type customListener struct {
	net.Listener
	serverSocket string
}

type customConn struct {
	net.Conn
	localAddr *net.UnixAddr
}

func (c *customConn) RemoteAddr() net.Addr {
	return c.localAddr
}

func (l *customListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return &customConn{
		Conn:      conn,
		localAddr: &net.UnixAddr{Net: "unix", Name: l.serverSocket},
	}, nil
}

func GetLocalIPAddress() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}

	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "127.0.0.1"
}
