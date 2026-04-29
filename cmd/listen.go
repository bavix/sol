package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/bavix/sol/internal/config"
	"github.com/bavix/sol/internal/deps"
	"github.com/bavix/sol/internal/domain/wol"
)

var errNoPorts = errors.New("at least one --port flag is required")

const portPartsCount = 2

var listenCmd = &cobra.Command{
	Use:   "listen",
	Short: "Listen for magic packets and trigger power action",
	Long:  "Listen for Wake-on-LAN magic packets on the specified network interface and trigger power action (shutdown/reboot) when received.",
	RunE: func(command *cobra.Command, _ []string) error {
		parsedRules, err := parsePorts()
		if err != nil {
			return err
		}

		cfg := &config.Config{
			InterfaceName: interfaceName,
			DryRun:        dryRun,
			Rules:         parsedRules,
		}

		builder := deps.NewBuilder(cfg)

		application, buildErr := builder.BuildListenService()
		if buildErr != nil {
			return buildErr
		}

		return application.Run(command.Context(), cfg.InterfaceName)
	},
}

var (
	interfaceName string
	dryRun        bool
	portStrings   []string
)

func init() {
	rootCmd.AddCommand(listenCmd)

	listenCmd.Flags().StringVar(&interfaceName, "iface", "", "Network interface name to bind to (required)")
	listenCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Log when a matching packet is received instead of executing the power action")
	listenCmd.Flags().StringArrayVar(&portStrings, "port", nil,
		"UDP port to listen on, optionally with action (e.g. '9' for shutdown, '8:reboot' for specific action). Can be specified multiple times")

	_ = listenCmd.MarkFlagRequired("iface")
}

func parsePorts() ([]wol.Rule, error) {
	if len(portStrings) == 0 {
		return nil, errNoPorts
	}

	defaultAction := wol.ActionShutdown

	rules := make([]wol.Rule, 0, len(portStrings))
	for _, ps := range portStrings {
		parts := strings.Split(ps, ":")
		portStr := parts[0]
		actionStr := ""

		if len(parts) == portPartsCount {
			actionStr = parts[1]
		}

		var port int
		if _, err := fmt.Sscanf(portStr, "%d", &port); err != nil {
			return nil, fmt.Errorf("invalid port %s: %w", portStr, err)
		}

		action := defaultAction

		if actionStr != "" {
			var err error

			action, err = wol.ParseAction(actionStr)
			if err != nil {
				return nil, fmt.Errorf("invalid action in port %s: %w", ps, err)
			}
		}

		rules = append(rules, wol.Rule{Port: port, Action: action})
	}

	return rules, nil
}
