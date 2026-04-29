package network

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUDPListenerReadPacket_ContextCanceled(t *testing.T) {
	t.Parallel()

	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	require.NoError(t, err)

	defer conn.Close()

	listener := &UDPListener{conn: conn}
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_, _, readErr := listener.ReadPacket(ctx)
	require.ErrorIs(t, readErr, context.Canceled)
}
