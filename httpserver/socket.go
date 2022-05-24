package httpserver

import (
	"context"
	"net"
)

// StreamSocket provides a listener for stream-oriented network connections.
type StreamSocket interface {
	Listen(ctx context.Context) (net.Listener, error)
}

// PacketSocket provides a listener for packet-oriented network connections.
type PacketSocket interface {
	Listen(ctx context.Context) (net.PacketConn, error)
}

type tcpSocket struct {
	address string
}

// TCP returns a StreamSocket for the given TCP address.
func TCP(address string) StreamSocket {
	return &tcpSocket{address}
}

func (l *tcpSocket) Listen(ctx context.Context) (net.Listener, error) {
	var lc net.ListenConfig
	return lc.Listen(ctx, "tcp", l.address)
}

type udpSocket struct {
	address string
}

// UDP returns a PacketSocket for the given UDP address.
func UDP(address string) PacketSocket {
	return &udpSocket{address}
}

func (l *udpSocket) Listen(ctx context.Context) (net.PacketConn, error) {
	var lc net.ListenConfig
	return lc.ListenPacket(ctx, "udp", l.address)
}
