//go:build !js

package blazewave

import "net"

func (nc *netConn) RemoteAddr() net.Addr {
	// Always return websocket address for WebSocket connections
	// regardless of the underlying connection type
	return websocketAddr{}
}

func (nc *netConn) LocalAddr() net.Addr {
	// Always return websocket address for WebSocket connections
	// regardless of the underlying connection type
	return websocketAddr{}
}
