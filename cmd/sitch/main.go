package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"charm.land/fang/v2"
	"github.com/spf13/cobra"
	"samouly.fun/sitch/internal/config"
	"samouly.fun/sitch/internal/render"
	"samouly.fun/sitch/internal/system"
)

var (
	version = "0.2.0"
	commit  = "unknown"
)

func main() {
	var noASCII, jsonOutput, truncate bool
	var colorMode, logo, logoFile string
	var configPath string
	root := &cobra.Command{
		Use:           "sitch",
		Short:         "a fast, beautiful system fetch",
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			info, err := system.Collect()
			if err != nil {
				return err
			}
			if colorMode != "" {
				if colorMode != "charmtone" && colorMode != "tty" {
					return fmt.Errorf("--color accepts charmtone or tty; custom colors must be defined in the TOML config")
				}
				cfg.ColorMode = colorMode
			}
			if jsonOutput || cfg.Format == "json" {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(info)
			}
			ascii := !noASCII
			if cfg.ASCII != nil {
				ascii = *cfg.ASCII
			}
			trunc := cfg.Truncate || truncate
			logoName := cfg.Logo
			if logo != "" {
				logoName = logo
			}
			logoPath := cfg.LogoFile
			if logoFile != "" {
				logoPath = logoFile
			}
			return render.PrintWithOptions(info, render.Options{
				Rows:         cfg.Rows,
				ColorMode:    cfg.ColorMode,
				Colors:       cfg.Colors,
				ASCII:        ascii,
				LogoPosition: cfg.LogoPosition,
				LogoJustify:  cfg.LogoJustify,
				LogoSize:     cfg.LogoSize,
				Logo:         logoName,
				LogoFile:     logoPath,
				Truncate:     trunc,
				FooterAlign:  cfg.FooterAlign,
			})
		},
	}
	root.Flags().BoolVarP(&noASCII, "no-ascii", "a", false, "hide the distribution logo")
	root.Flags().BoolVarP(&jsonOutput, "json", "j", false, "output system information as JSON")
	root.Flags().StringVarP(&configPath, "config", "c", "", "path to TOML configuration")
	root.Flags().StringVar(&colorMode, "color", "", "color mode: charmtone, tty, or custom")
	root.Flags().StringVar(&logo, "logo", "", "bundled logo id to display instead of the detected distro")
	root.Flags().StringVar(&logoFile, "logo-file", "", "path to a custom ASCII art file")
	root.Flags().BoolVar(&truncate, "truncate", false, "clip the logo vertically so the body never grows taller than the grid+header")
	root.Flags().BoolP("fetch", "f", false, "print system information (default)")

	if err := fang.Execute(context.Background(), root, fang.WithVersion(version), fang.WithCommit(commit)); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
