package cmd

import (
	"log"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "sol",
	Short: "Shutdown-on-LAN service",
	Long:  "sol is a service that listens for Wake-on-LAN magic packets and shuts down the system when received.",
}

// Execute executes the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func init() {
	// Add any global flags here if needed
}
