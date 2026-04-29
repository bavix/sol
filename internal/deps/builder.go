package deps

import (
	"fmt"
	"sync"

	"github.com/bavix/sol/internal/app"
	"github.com/bavix/sol/internal/config"
	"github.com/bavix/sol/internal/domain/wol"
	"github.com/bavix/sol/internal/infra/network"
	"github.com/bavix/sol/internal/infra/system"
)

type Builder struct {
	cfg *config.Config

	resolverOnce sync.Once
	resolver     app.InterfaceResolver

	factoryOnce sync.Once
	factory     app.PacketListenerFactory

	powerOnce sync.Once
	power     wol.PowerController

	listenOnce sync.Once
	listen     *app.ListenService
}

func NewBuilder(cfg *config.Config) *Builder {
	return &Builder{cfg: cfg}
}

func (b *Builder) InterfaceResolver() app.InterfaceResolver { //nolint:ireturn
	b.resolverOnce.Do(func() {
		b.resolver = network.NewInterfaceResolver()
	})

	return b.resolver
}

func (b *Builder) PacketListenerFactory() app.PacketListenerFactory { //nolint:ireturn
	b.factoryOnce.Do(func() {
		b.factory = network.NewUDPListenerFactory()
	})

	return b.factory
}

func (b *Builder) PowerController() wol.PowerController { //nolint:ireturn
	b.powerOnce.Do(func() {
		b.power = system.NewPowerController()
	})

	return b.power
}

func (b *Builder) BuildListenService() (*app.ListenService, error) {
	var buildErr error

	b.listenOnce.Do(func() {
		ifaceIP, ifaceMAC, err := b.InterfaceResolver().Resolve(b.cfg.InterfaceName)
		if err != nil {
			buildErr = fmt.Errorf("failed to get IP/MAC for interface %q: %w", b.cfg.InterfaceName, err)

			return
		}

		policy, err := wol.NewRoutingPolicy(b.cfg.Rules, ifaceMAC)
		if err != nil {
			buildErr = err

			return
		}

		b.listen = app.NewListenService(
			b.InterfaceResolver(),
			b.PacketListenerFactory(),
			b.PowerController(),
			policy,
			ifaceMAC,
			ifaceIP,
			b.cfg.DryRun,
		)
	})

	return b.listen, buildErr
}
