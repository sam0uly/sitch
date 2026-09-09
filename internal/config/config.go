// Package config loads Sitch's optional TOML configuration.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// Config controls output format, visible specs, and their row grouping.
type Config struct {
	Format        string       `toml:"format"`
	ASCII         *bool        `toml:"ascii"`
	ColorMode     string       `toml:"color_mode"`
	LogoPosition  string       `toml:"logo_position"`
	LogoJustify   string       `toml:"logo_justify"`
	LogoSize      string       `toml:"logo_size"`
	LogoColorMode string       `toml:"logo_color_mode"`
	LogoColor     string       `toml:"logo_color"`
	Logo          string       `toml:"logo"`
	LogoFile      string       `toml:"logo_file"`
	Truncate      bool         `toml:"truncate"`
	FooterAlign   string       `toml:"footer_align"`
	Colors        CustomColors `toml:"colors"`
	Rows          [][]string   `toml:"rows"`
}

// CustomColors contains the complete palette required by custom mode.
type CustomColors struct {
	Border     string            `toml:"border"`
	Background string            `toml:"background"`
	Header     string            `toml:"header"`
	Value      string            `toml:"value"`
	Sitch      string            `toml:"sitch"`
	Titles     map[string]string `toml:"titles"`
}

// Default returns the untouched default Sitch layout.
func Default() Config {
	return Config{Format: "terminal", ColorMode: "charmtone", Rows: [][]string{
		{"os", "host"},
		{"disk", "memory"},
		{"kernel", "uptime"},
		{"gpu", "igpu"},
		{"cpu", "packages"},
		{"shell", "display"},
		{"desktop", "terminal"},
		{"os-age", "locale"},
	}}
}

// Load reads path, or returns defaults when path is empty or absent.
// On first launch with no explicit --config flag it also writes a default
// config file to the user's config directory so the user can discover it.
func Load(path string) (Config, error) {
	cfg := Default()
	if path == "" {
		path = defaultPath()
	}
	if path == "" {
		return cfg, nil
	}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if err := writeDefaultConfig(path); err != nil {
			return Default(), nil
		}
		return cfg, nil
	} else if err != nil {
		return Config{}, err
	}
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return Config{}, err
	}
	if cfg.Format == "" {
		cfg.Format = "terminal"
	}
	if cfg.ColorMode == "" {
		cfg.ColorMode = "charmtone"
	}
	if len(cfg.Rows) == 0 {
		cfg.Rows = Default().Rows
	}
	if cfg.Format != "terminal" && cfg.Format != "json" {
		return Config{}, fmt.Errorf("format must be terminal or json, got %q", cfg.Format)
	}
	if cfg.ColorMode != "charmtone" && cfg.ColorMode != "tty" && cfg.ColorMode != "custom" {
		return Config{}, fmt.Errorf("color_mode must be charmtone, tty, or custom, got %q", cfg.ColorMode)
	}
	if cfg.FooterAlign != "" && cfg.FooterAlign != "full" && cfg.FooterAlign != "grid" {
		return Config{}, fmt.Errorf("footer_align must be full or grid, got %q", cfg.FooterAlign)
	}
	if cfg.LogoPosition != "" && cfg.LogoPosition != "left" && cfg.LogoPosition != "right" && cfg.LogoPosition != "top" && cfg.LogoPosition != "bottom" {
		return Config{}, fmt.Errorf("logo_position must be left, right, top, or bottom, got %q", cfg.LogoPosition)
	}
	if cfg.LogoJustify != "" && cfg.LogoJustify != "top" && cfg.LogoJustify != "middle" && cfg.LogoJustify != "bottom" {
		return Config{}, fmt.Errorf("logo_justify must be top, middle, or bottom, got %q", cfg.LogoJustify)
	}
	if cfg.LogoSize != "" && cfg.LogoSize != "regular" && cfg.LogoSize != "small" {
		return Config{}, fmt.Errorf("logo_size must be regular or small, got %q", cfg.LogoSize)
	}
	if cfg.LogoColorMode != "" && cfg.LogoColorMode != "multi" && cfg.LogoColorMode != "single" {
		return Config{}, fmt.Errorf("logo_color_mode must be multi or single, got %q", cfg.LogoColorMode)
	}
	if strings.TrimSpace(cfg.LogoColor) != "" && !isColorValue(cfg.LogoColor) {
		return Config{}, fmt.Errorf("logo_color %q is invalid; valid colors are hex (#rgb, #rrggbb), ANSI (0-255), or terminal names", cfg.LogoColor)
	}
	if cfg.LogoFile != "" {
		if _, err := os.Stat(cfg.LogoFile); err != nil {
			return Config{}, fmt.Errorf("logo_file %q: %w", cfg.LogoFile, err)
		}
	}
	if cfg.ColorMode == "custom" {
		if cfg.Colors.Titles == nil {
			cfg.Colors.Titles = map[string]string{}
		}
		if err := validateCustomColorFormat(cfg.Colors); err != nil {
			return Config{}, err
		}
	}
	for rowIndex, row := range cfg.Rows {
		if len(row) == 0 {
			return Config{}, fmt.Errorf("row %d is empty", rowIndex+1)
		}
		if len(row) > 3 {
			return Config{}, fmt.Errorf("row %d has %d columns; at most 3 are supported", rowIndex+1, len(row))
		}
		for columnIndex, spec := range row {
			if !supportedSpecs[strings.ToLower(spec)] {
				return Config{}, fmt.Errorf("unknown spec %q at row %d column %d", spec, rowIndex+1, columnIndex+1)
			}
			cfg.Rows[rowIndex][columnIndex] = strings.ToLower(spec)
		}
	}
	return cfg, nil
}

// validateCustomColorFormat ensures custom colors that are set parse as
// colors but deliberately treats empty strings as unset. Empty entries are
// rendered transparent (no styling) rather than rejected.
func validateCustomColorFormat(colors CustomColors) error {
	invalid := make([]string, 0, len(supportedSpecs)+5)
	for name, value := range map[string]string{
		"border": colors.Border, "background": colors.Background, "header": colors.Header,
		"value": colors.Value, "sitch": colors.Sitch,
	} {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if !isColorValue(trimmed) {
			invalid = append(invalid, "colors."+name)
		}
	}
	titles := make([]string, 0, len(colors.Titles))
	for spec := range colors.Titles {
		titles = append(titles, spec)
	}
	for _, spec := range titles {
		trimmed := strings.TrimSpace(colors.Titles[spec])
		if trimmed == "" {
			continue
		}
		if !isColorValue(trimmed) {
			invalid = append(invalid, "colors.titles."+spec)
		}
	}
	if len(invalid) > 0 {
		invalid = uniqueSorted(invalid)
		return fmt.Errorf("custom color mode has invalid colors: %s\n\nValid colors are hex (#rgb, #rrggbb), ANSI (0-255), or terminal names; empty values are transparent. A complete example:\n\n%s", strings.Join(invalid, ", "), CustomColorExample())
	}
	return nil
}

// isColorValue accepts empty strings and the color spellings lipgloss can
// parse: hex, ANSI numbers, ordinary color names, or "none"/"transparent".
func isColorValue(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	switch lower {
	case "", "none", "transparent", "default", "reset":
		return true
	}
	if strings.HasPrefix(lower, "#") {
		hex := lower[1:]
		if len(hex) != 3 && len(hex) != 6 {
			return false
		}
		for _, r := range hex {
			if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
				return false
			}
		}
		return true
	}
	if number, ok := parseColorNumber(lower); ok {
		return number >= 0 && number <= 255
	}
	switch lower {
	case "black", "red", "green", "yellow", "blue", "magenta", "cyan", "white",
		"brightblack", "brightred", "brightgreen", "brightyellow",
		"brightblue", "brightmagenta", "brightcyan", "brightwhite",
		"lightblack", "lightred", "lightgreen", "lightyellow",
		"lightblue", "lightmagenta", "lightcyan", "lightwhite":
		return true
	default:
		return false
	}
}

func parseColorNumber(value string) (int, bool) {
	sign := 1
	rest := value
	if strings.HasPrefix(rest, "+") {
		rest = rest[1:]
	} else if strings.HasPrefix(rest, "-") {
		sign = -1
		rest = rest[1:]
	}
	if rest == "" {
		return 0, false
	}
	number := 0
	for _, r := range rest {
		if r < '0' || r > '9' {
			return 0, false
		}
		number = number*10 + int(r-'0')
	}
	return sign * number, true
}

func uniqueSorted(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// CustomColorExample returns a complete custom palette users can edit.
func CustomColorExample() string {
	return `color_mode = "custom"

[colors]
border = "#8a8a8a"
background = "#1f1f28"
header = "#c0caf5"
value = "#f8f8f2"
sitch = "#bb9af7"

[colors.titles]
os = "#ff9e64"
host = "#e0af68"
disk = "#f7768e"
memory = "#9ece6a"
kernel = "#7dcfff"
uptime = "#bb9af7"
gpu = "#2ac3de"
igpu = "#73daca"
cpu = "#ff7a93"
packages = "#c0caf5"
shell = "#f6c177"
display = "#7aa2f7"
desktop = "#a6da95"
terminal = "#e0af68"
os-age = "#f7768e"
locale = "#c6a0f6"
`
}

var supportedSpecs = map[string]bool{
	"os": true, "host": true, "disk": true, "memory": true, "kernel": true,
	"uptime": true, "gpu": true, "igpu": true, "cpu": true, "packages": true,
	"shell": true, "display": true, "desktop": true,
	"terminal": true, "os-age": true, "locale": true,
}

func defaultPath() string {
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, "sitch", "config.toml")
	}
	return ""
}

// DefaultTOML renders the default config as a TOML string for first-launch files.
func DefaultTOML() string {
	return `# Default Sitch configuration. Edit and save; sitch will pick it up on the next run.
# See https://samouly.fun/sitch for the full list of options.
format = "terminal"
color_mode = "charmtone"

# logo_color_mode = "multi"  # "multi" cycles the theme palette; "single" paints the whole logo with logo_color
# logo_color = "#ff985a"

rows = [
  ["os", "host"],
  ["disk", "memory"],
  ["kernel", "uptime"],
  ["gpu", "igpu"],
  ["cpu", "packages"],
  ["shell", "display"],
  ["desktop", "terminal"],
  ["os-age", "locale"],
]
`
}

func writeDefaultConfig(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(DefaultTOML()), 0o600)
}
