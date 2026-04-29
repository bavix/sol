package config

import "github.com/bavix/sol/internal/domain/wol"

// Config holds the application configuration.
type Config struct {
	InterfaceName string
	DryRun        bool
	Rules         []wol.Rule
}
