package wol

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrUnknownAction = errors.New("unknown action: must be shutdown or reboot")
	ErrInvalidRule   = errors.New("invalid rule format: expected port:action")
	ErrDuplicatePort = errors.New("duplicate port in rules")
)

type Action string

const (
	ActionShutdown Action = "shutdown"
	ActionReboot   Action = "reboot"
)

func ParseAction(s string) (Action, error) {
	switch strings.ToLower(s) {
	case "shutdown", "s":
		return ActionShutdown, nil
	case "reboot", "r":
		return ActionReboot, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnknownAction, s)
	}
}

type Rule struct {
	Port   int
	Action Action
}

type RoutingPolicy struct {
	rules     []Rule
	fallback  Rule
	targetMAC []byte
}

func NewRoutingPolicy(rules []Rule, targetMAC []byte) (*RoutingPolicy, error) {
	seen := make(map[int]bool)
	for _, r := range rules {
		if seen[r.Port] {
			return nil, fmt.Errorf("%w: %d", ErrDuplicatePort, r.Port)
		}

		seen[r.Port] = true
	}

	policy := &RoutingPolicy{
		rules:     rules,
		targetMAC: targetMAC,
	}

	if len(rules) > 0 {
		policy.fallback = rules[0]
	}

	return policy, nil
}

func (p *RoutingPolicy) Match(payload []byte, dstPort int) (Action, bool) {
	expected := BuildMagicPacket(p.targetMAC)
	if !bytes.Contains(payload, expected) {
		return "", false
	}

	for _, r := range p.rules {
		if r.Port == dstPort {
			return r.Action, true
		}
	}

	return "", false
}

func (p *RoutingPolicy) Ports() []int {
	ports := make([]int, len(p.rules))
	for i, r := range p.rules {
		ports[i] = r.Port
	}

	return ports
}

func (p *RoutingPolicy) Rules() []Rule {
	return p.rules
}

type PowerController interface {
	Execute(ctx context.Context, action Action) error
}
