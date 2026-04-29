package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"

	"github.com/bavix/sol/internal/domain/wol"
)

var errNilListener = errors.New("nil listener")

const packetChannelSize = 2

type packet struct {
	payload []byte
	src     *net.UDPAddr
	port    int
}

type InterfaceResolver interface {
	Resolve(name string) (net.IP, net.HardwareAddr, error)
}

type PacketListener interface {
	ReadPacket(ctx context.Context) ([]byte, *net.UDPAddr, error)
	Close() error
}

type PacketListenerFactory interface {
	Create(port int) (PacketListener, error)
}

type ListenService struct {
	resolver InterfaceResolver
	factory  PacketListenerFactory
	power    wol.PowerController
	policy   *wol.RoutingPolicy
	ifaceMAC net.HardwareAddr
	ifaceIP  net.IP
	dryRun   bool
}

func NewListenService(
	resolver InterfaceResolver,
	factory PacketListenerFactory,
	power wol.PowerController,
	policy *wol.RoutingPolicy,
	ifaceMAC net.HardwareAddr,
	ifaceIP net.IP,
	dryRun bool,
) *ListenService {
	return &ListenService{
		resolver: resolver,
		factory:  factory,
		power:    power,
		policy:   policy,
		ifaceMAC: ifaceMAC,
		ifaceIP:  ifaceIP,
		dryRun:   dryRun,
	}
}

func (s *ListenService) Run(ctx context.Context, interfaceName string) error {
	log.Printf("Using interface %q: IP=%s, MAC=%s", interfaceName, s.ifaceIP, s.ifaceMAC)
	s.logRules()

	listeners, err := s.createListeners()
	if err != nil {
		return err
	}
	defer s.closeListeners(listeners)

	pktCh, errCh := s.startListenerGoroutines(ctx, listeners)

	s.eventLoop(ctx, pktCh, errCh)

	return nil
}

func (s *ListenService) logRules() {
	rules := s.policy.Rules()
	for i, rule := range rules {
		log.Printf("Rule %d: port=%d action=%s", i+1, rule.Port, rule.Action)
	}
}

func (s *ListenService) createListeners() ([]PacketListener, error) {
	ports := s.policy.Ports()
	listeners := make([]PacketListener, 0, len(ports))

	for _, p := range ports {
		lis, err := s.factory.Create(p)
		if err != nil {
			for _, l := range listeners {
				_ = l.Close()
			}

			return nil, fmt.Errorf("failed to create listener on port %d: %w", p, err)
		}

		listeners = append(listeners, lis)
	}

	for _, p := range ports {
		log.Printf("Listening on 0.0.0.0:%d", p)
	}

	return listeners, nil
}

func (s *ListenService) closeListeners(listeners []PacketListener) {
	for _, l := range listeners {
		_ = l.Close()
	}
}

func (s *ListenService) startListenerGoroutines(ctx context.Context, listeners []PacketListener) (chan packet, chan error) {
	ports := s.policy.Ports()
	pktCh := make(chan packet, packetChannelSize)
	errCh := make(chan error, len(listeners))

	for idx, lis := range listeners {
		go s.listenOnPort(ctx, lis, ports[idx], pktCh, errCh)
	}

	return pktCh, errCh
}

func (s *ListenService) listenOnPort(ctx context.Context, listener PacketListener, port int, pktCh chan packet, errCh chan error) {
	if listener == nil {
		errCh <- errNilListener

		return
	}

	for {
		payload, src, err := listener.ReadPacket(ctx)
		if shouldStop(err) {
			errCh <- err

			return
		}

		if err != nil {
			log.Printf("read error on port %d: %v", port, err)

			continue
		}

		select {
		case pktCh <- packet{payload, src, port}:
		case <-ctx.Done():
			errCh <- ctx.Err()

			return
		}
	}
}

func (s *ListenService) eventLoop(ctx context.Context, pktCh chan packet, errCh chan error) {
	for {
		select {
		case <-ctx.Done():
			return
		case err := <-errCh:
			if shouldStop(err) {
				return
			}

			log.Printf("listener error: %v", err)
		case pkt := <-pktCh:
			s.handlePacket(ctx, pkt)
		}
	}
}

func (s *ListenService) handlePacket(ctx context.Context, pkt packet) {
	action, matched := s.policy.Match(pkt.payload, pkt.port)
	if !matched {
		log.Printf("Non-matching packet from %s, port=%d, len=%d", pkt.src, pkt.port, len(pkt.payload))

		return
	}

	trigger := ternary(s.dryRun, "DRY-RUN", string(action))
	log.Printf("Magic packet match from %s, port=%d, action=%s - triggering %s",
		pkt.src, pkt.port, action, trigger)

	if s.dryRun {
		return
	}

	if execErr := s.power.Execute(ctx, action); execErr != nil {
		log.Printf("%s failed: %v", action, execErr)
	}
}

func shouldStop(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

func ternary(cond bool, a string, b string) string {
	if cond {
		return a
	}

	return b
}
