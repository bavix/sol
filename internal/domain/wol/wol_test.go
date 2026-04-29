package wol_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/sol/internal/domain/wol"
)

func TestParseAction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected wol.Action
		wantErr  bool
	}{
		{input: "shutdown", expected: wol.ActionShutdown, wantErr: false},
		{input: "Shutdown", expected: wol.ActionShutdown, wantErr: false},
		{input: "s", expected: wol.ActionShutdown, wantErr: false},
		{input: "S", expected: wol.ActionShutdown, wantErr: false},
		{input: "reboot", expected: wol.ActionReboot, wantErr: false},
		{input: "Reboot", expected: wol.ActionReboot, wantErr: false},
		{input: "r", expected: wol.ActionReboot, wantErr: false},
		{input: "R", expected: wol.ActionReboot, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			got, err := wol.ParseAction(tt.input)
			require.NoError(t, err)
			require.Equal(t, tt.expected, got)
		})
	}

	t.Run("invalid poweroff", func(t *testing.T) {
		t.Parallel()

		_, err := wol.ParseAction("poweroff")
		require.Error(t, err)
	})

	t.Run("invalid empty", func(t *testing.T) {
		t.Parallel()

		_, err := wol.ParseAction("")
		require.Error(t, err)
	})
}

func TestNewRoutingPolicy(t *testing.T) {
	t.Parallel()

	mac := []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}

	t.Run("single rule", func(t *testing.T) {
		t.Parallel()

		_, err := wol.NewRoutingPolicy([]wol.Rule{
			{Port: 9, Action: wol.ActionShutdown},
		}, mac)
		require.NoError(t, err)
	})

	t.Run("multiple rules different ports", func(t *testing.T) {
		t.Parallel()

		_, err := wol.NewRoutingPolicy([]wol.Rule{
			{Port: 9, Action: wol.ActionShutdown},
			{Port: 8, Action: wol.ActionReboot},
		}, mac)
		require.NoError(t, err)
	})

	t.Run("duplicate port", func(t *testing.T) {
		t.Parallel()

		_, err := wol.NewRoutingPolicy([]wol.Rule{
			{Port: 9, Action: wol.ActionShutdown},
			{Port: 9, Action: wol.ActionReboot},
		}, mac)
		require.Error(t, err)
	})
}

func TestRoutingPolicyMatch(t *testing.T) {
	t.Parallel()

	mac := []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}
	magic := wol.BuildMagicPacket(mac)

	policy := mustNewRoutingPolicy(t, []wol.Rule{
		{Port: 9, Action: wol.ActionShutdown},
		{Port: 8, Action: wol.ActionReboot},
	}, mac)

	testRoutingPolicyMatchCases(t, policy, magic)
}

func testRoutingPolicyMatchCases(t *testing.T, policy *wol.RoutingPolicy, magic []byte) {
	t.Helper()

	tests := []struct {
		name      string
		payload   []byte
		port      int
		wantAct   wol.Action
		wantMatch bool
	}{
		{
			name:      "shutdown port match",
			payload:   magic,
			port:      9,
			wantAct:   wol.ActionShutdown,
			wantMatch: true,
		},
		{
			name:      "reboot port match",
			payload:   magic,
			port:      8,
			wantAct:   wol.ActionReboot,
			wantMatch: true,
		},
		{
			name:      "no match wrong port",
			payload:   magic,
			port:      7,
			wantMatch: false,
		},
		{
			name:      "no match wrong payload",
			payload:   []byte("not-magic"),
			port:      9,
			wantMatch: false,
		},
		{
			name:      "action matches on port 9",
			payload:   magic,
			port:      9,
			wantAct:   wol.ActionShutdown,
			wantMatch: true,
		},
		{
			name:      "action matches on port 8",
			payload:   magic,
			port:      8,
			wantAct:   wol.ActionReboot,
			wantMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			act, matched := policy.Match(tt.payload, tt.port)
			require.Equal(t, tt.wantMatch, matched, "Match() matched = %v, want %v", matched, tt.wantMatch)
			require.Equal(t, tt.wantAct, act, "Match() action = %v, want %v", act, tt.wantAct)
		})
	}
}

func TestRoutingPolicyPorts(t *testing.T) {
	t.Parallel()

	mac := []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}
	policy := mustNewRoutingPolicy(t, []wol.Rule{
		{Port: 9, Action: wol.ActionShutdown},
		{Port: 8, Action: wol.ActionReboot},
	}, mac)

	ports := policy.Ports()
	require.Len(t, ports, 2)
}

func TestRoutingPolicyRules(t *testing.T) {
	t.Parallel()

	mac := []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}
	rules := []wol.Rule{
		{Port: 9, Action: wol.ActionShutdown},
		{Port: 8, Action: wol.ActionReboot},
	}
	policy := mustNewRoutingPolicy(t, rules, mac)

	returnedRules := policy.Rules()
	require.Len(t, returnedRules, len(rules))
}

func TestBuildMagicPacket(t *testing.T) {
	t.Parallel()

	mac := []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}
	pkt := wol.BuildMagicPacket(mac)

	expectedLen := 6 + 16*6
	require.Len(t, pkt, expectedLen)

	for i, b := range pkt[:6] {
		require.Equal(t, byte(0xFF), b, "header byte %d", i)
	}
}

func TestContainsMagicPacket(t *testing.T) {
	t.Parallel()

	mac := []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}
	magic := wol.BuildMagicPacket(mac)

	tests := []struct {
		name string
		got  []byte
		want bool
	}{
		{"exact", magic, true},
		{"with prefix", append([]byte{0x00, 0x01}, magic...), true},
		{"with suffix", append(magic, 0x03, 0x04), true},
		{"wrong mac", wol.BuildMagicPacket([]byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66}), false},
		{"too short", []byte{0xFF, 0xFF, 0xFF}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, wol.ContainsMagicPacket(tt.got, magic))
		})
	}
}

func mustNewRoutingPolicy(t *testing.T, rules []wol.Rule, mac []byte) *wol.RoutingPolicy {
	t.Helper()

	policy, err := wol.NewRoutingPolicy(rules, mac)
	require.NoError(t, err)

	return policy
}
