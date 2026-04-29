package network

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/bavix/sol/internal/app"
	"github.com/bavix/sol/internal/domain/wol"
)

type UDPListenerFactory struct{}

func NewUDPListenerFactory() *UDPListenerFactory {
	return &UDPListenerFactory{}
}

func (f *UDPListenerFactory) Create(port int) (app.PacketListener, error) { //nolint:ireturn
	addr := &net.UDPAddr{IP: net.IPv4zero, Port: port}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to bind UDP %s:%d: %w", addr.IP.String(), port, err)
	}

	return &UDPListener{conn: conn}, nil
}

type UDPListener struct {
	conn *net.UDPConn
}

func (l *UDPListener) ReadPacket(ctx context.Context) ([]byte, *net.UDPAddr, error) {
	return l.readPacket(ctx)
}

func (l *UDPListener) Close() error {
	return l.conn.Close()
}

func (l *UDPListener) readPacket(ctx context.Context) ([]byte, *net.UDPAddr, error) {
	buf := make([]byte, wol.BufferSize)

	for {
		if deadlineErr := l.conn.SetReadDeadline(time.Now().Add(time.Second)); deadlineErr != nil {
			return nil, nil, deadlineErr
		}

		n, src, err := l.conn.ReadFromUDP(buf)
		if err == nil {
			return buf[:n], src, nil
		}

		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			select {
			case <-ctx.Done():
				return nil, nil, ctx.Err()
			default:
				continue
			}
		}

		return nil, nil, err
	}
}
