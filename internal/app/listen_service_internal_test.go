package app

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/sol/internal/domain/wol"
)

type resolverMock struct {
	ip  net.IP
	mac net.HardwareAddr
	err error
}

func (m *resolverMock) Resolve(_ string) (net.IP, net.HardwareAddr, error) {
	if m.err != nil {
		return nil, nil, m.err
	}

	return m.ip, m.mac, nil
}

type factoryMock struct {
	listener PacketListener
	err      error
}

func (m *factoryMock) Create(_ int) (PacketListener, error) {
	if m.err != nil {
		return nil, m.err
	}

	return m.listener, nil
}

type powerMock struct {
	calls  int
	action wol.Action
	err    error
}

func (m *powerMock) Execute(_ context.Context, action wol.Action) error {
	m.calls++
	m.action = action

	return m.err
}

type listenerMock struct {
	packets [][]byte
	srcs    []*net.UDPAddr
	errs    []error
	idx     int
	closed  bool
}

func (m *listenerMock) ReadPacket(_ context.Context) ([]byte, *net.UDPAddr, error) {
	if m.idx >= len(m.packets) {
		return nil, nil, context.Canceled
	}

	err := m.errs[m.idx]
	if err != nil {
		m.idx++

		return nil, nil, err
	}

	payload := m.packets[m.idx]
	src := m.srcs[m.idx]
	m.idx++

	return payload, src, nil
}

func (m *listenerMock) Close() error {
	m.closed = true

	return nil
}

type testFixture struct {
	listener *listenerMock
	power    *powerMock
	service  *ListenService
}

func newFixture(t *testing.T, port int, action wol.Action, packets [][]byte, packetErrs []error) *testFixture {
	t.Helper()

	mac := net.HardwareAddr{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}

	policy, err := wol.NewRoutingPolicy([]wol.Rule{{Port: port, Action: action}}, mac)
	require.NoError(t, err)

	srcs := make([]*net.UDPAddr, len(packets))
	for i := range packets {
		srcs[i] = &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 10001 + i}
	}

	listener := &listenerMock{
		packets: packets,
		srcs:    srcs,
		errs:    packetErrs,
	}
	power := &powerMock{}

	svc := NewListenService(
		&resolverMock{ip: net.IPv4(127, 0, 0, 1), mac: mac},
		&factoryMock{listener: listener},
		power,
		policy,
		mac,
		net.IPv4(127, 0, 0, 1),
		false,
	)

	return &testFixture{
		listener: listener,
		power:    power,
		service:  svc,
	}
}

func TestListenServiceRun_MagicPacketShutdown(t *testing.T) {
	t.Parallel()

	mac := net.HardwareAddr{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}
	magic := wol.BuildMagicPacket(mac)

	fixture := newFixture(t, 9, wol.ActionShutdown, [][]byte{magic}, []error{nil, context.Canceled})

	err := fixture.service.Run(context.Background(), "en0")
	require.NoError(t, err)

	require.Equal(t, 1, fixture.power.calls)
	require.Equal(t, wol.ActionShutdown, fixture.power.action)
	require.True(t, fixture.listener.closed)
}

func TestListenServiceRun_MagicPacketReboot(t *testing.T) {
	mac := net.HardwareAddr{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}
	magic := wol.BuildMagicPacket(mac)

	fixture := newFixture(t, 8, wol.ActionReboot, [][]byte{magic}, []error{nil, context.Canceled})

	err := fixture.service.Run(context.Background(), "en0")
	require.NoError(t, err)

	require.Equal(t, 1, fixture.power.calls)
	require.Equal(t, wol.ActionReboot, fixture.power.action)
}

func TestListenServiceRun_DryRunSkipsPowerAction(t *testing.T) {
	t.Parallel()

	mac := net.HardwareAddr{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}
	magic := wol.BuildMagicPacket(mac)

	fixture := newFixture(t, 9, wol.ActionShutdown, [][]byte{magic}, []error{context.Canceled})

	err := fixture.service.Run(context.Background(), "en0")
	require.NoError(t, err)
	require.Equal(t, 0, fixture.power.calls)
}

func TestListenServiceRun_ContextCanceled(t *testing.T) {
	t.Parallel()

	mac := net.HardwareAddr{1, 2, 3, 4, 5, 6}

	policy, err := wol.NewRoutingPolicy([]wol.Rule{{Port: 9, Action: wol.ActionShutdown}}, mac)
	require.NoError(t, err)

	listener := &listenerMock{
		packets: [][]byte{},
		srcs:    []*net.UDPAddr{},
		errs:    []error{context.Canceled},
	}

	svc := NewListenService(
		&resolverMock{ip: net.IPv4(127, 0, 0, 1), mac: mac},
		&factoryMock{listener: listener},
		&powerMock{},
		policy,
		mac,
		net.IPv4(127, 0, 0, 1),
		false,
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = svc.Run(ctx, "en0")
	require.NoError(t, err)
}

func TestListenServiceRun_DuplicatePortRules(t *testing.T) {
	t.Parallel()

	mac := []byte{1, 2, 3, 4, 5, 6}

	_, err := wol.NewRoutingPolicy([]wol.Rule{
		{Port: 9, Action: wol.ActionShutdown},
		{Port: 9, Action: wol.ActionReboot},
	}, mac)
	require.Error(t, err)
}
