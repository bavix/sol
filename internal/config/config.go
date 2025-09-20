package config

import (
	"flag"
)

// Config holds the application configuration.
type Config struct {
	InterfaceName string
	Port          int
	DryRun        bool
}

// LoadConfig parses command line flags and returns the configuration.
func LoadConfig() *Config {
	cfg := &Config{}
	
	flag.StringVar(&cfg.InterfaceName, "iface", "", "Network interface name to bind to (e.g., 'Ethernet 4', 'eth0')")
	flag.IntVar(&cfg.Port, "port", 9, "UDP port to listen on (7 or 9 are common for WoL)")
	flag.BoolVar(&cfg.DryRun, "dry-run", false, "Log when a matching packet is received instead of shutting down")
	
	return cfg
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.InterfaceName == "" {
		return flag.ErrHelp
	}
	return nil
}
