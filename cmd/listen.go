package cmd

import (
	"github.com/spf13/cobra"

	"github.com/bavix/sol/internal/app"
	"github.com/bavix/sol/internal/config"
)

var listenCmd = &cobra.Command{
	Use:   "listen",
	Short: "Listen for magic packets and shutdown system",
	Long:  "Listen for Wake-on-LAN magic packets on the specified network interface and shutdown the system when received.",
	RunE: func(_ *cobra.Command, _ []string) error {
		cfg := &config.Config{
			InterfaceName: interfaceName,
			Port:          port,
			DryRun:        dryRun,
		}

		application := app.New(cfg)

		return application.Run()
	},
}

var (
	interfaceName string
	port          int
	dryRun        bool
)

func init() {
	rootCmd.AddCommand(listenCmd)

	listenCmd.Flags().StringVar(&interfaceName, "iface", "", "Network interface name to bind to (e.g., 'Ethernet 4', 'eth0')")
	listenCmd.Flags().IntVar(&port, "port", app.WoLPortDefault, "UDP port to listen on (7 or 9 are common for WoL)")
	listenCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Log when a matching packet is received instead of shutting down")

	_ = listenCmd.MarkFlagRequired("iface")
}
