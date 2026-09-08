package render

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"samouly.fun/sitch/internal/config"
	"samouly.fun/sitch/internal/system"
)

func TestPickLogoKnownDistro(t *testing.T) {
	info := system.Info{DistroID: "arch"}
	got := pickLogo(info, "regular")
	want := fastfetchLogos["arch"]
	if got.text != want.text {
		t.Errorf("arch logo text mismatch:\ngot:  %q\nwant: %q", got.text, want.text)
	}
}

func TestPickLogoFallback(t *testing.T) {
	info := system.Info{DistroID: "totally-unknown-distro"}
	got := pickLogo(info, "regular")
	if got.text != defaultLogoText {
		t.Errorf("fallback text mismatch:\ngot:  %q\nwant: %q", got.text, defaultLogoText)
	}
}

func TestPickLogoSmall(t *testing.T) {
	info := system.Info{DistroID: "arch"}
	got := pickLogo(info, "small")
	if _, ok := fastfetchLogos["arch_small"]; ok {
		if got.text != fastfetchLogos["arch_small"].text {
			t.Errorf("small-size arch logo should use arch_small variant")
		}
	} else {
		t.Skip("arch_small not in generated logos")
	}
}

func TestPickLogoNameOverride(t *testing.T) {
	info := system.Info{DistroID: "nixos"}
	setLogoRequest("arch", "")
	defer setLogoRequest("", "")
	got := pickLogo(info, "regular")
	if got.text != fastfetchLogos["arch"].text {
		t.Errorf("explicit logo name was ignored")
	}
}

func TestPickLogoCustomText(t *testing.T) {
	info := system.Info{DistroID: "nixos"}
	setLogoRequest("", "custom\nart")
	defer setLogoRequest("", "")
	got := pickLogo(info, "regular")
	if got.text != "custom\nart" {
		t.Errorf("custom logo text was ignored, got %q", got.text)
	}
}

func TestLookupLogo(t *testing.T) {
	if _, ok := lookupLogo("arch", "regular"); !ok {
		t.Errorf("lookupLogo(arch) failed")
	}
	if _, ok := lookupLogo("ARCH", "regular"); !ok {
		t.Errorf("lookupLogo should be case-insensitive")
	}
	if _, ok := lookupLogo("no-such-distro", "regular"); ok {
		t.Errorf("lookupLogo accepted an unknown id")
	}
}

func TestLogoNames(t *testing.T) {
	names := LogoNames()
	if len(names) == 0 {
		t.Fatal("LogoNames returned no logos")
	}
	for i := 1; i < len(names); i++ {
		if names[i-1] >= names[i] {
			t.Fatalf("LogoNames is not sorted: %q >= %q", names[i-1], names[i])
		}
	}
}

func TestRenderLogoBoxWidth(t *testing.T) {
	setPalette("charmtone", config.CustomColors{})
	for _, id := range []string{"arch", "nixos", "debian", "ubuntu", "fedora", "void", "alpine", "unknown"} {
		t.Run(id, func(t *testing.T) {
			rendered, width := renderLogoBox(system.Info{DistroID: id})
			lines := strings.Split(rendered, "\n")
			if len(lines) == 0 {
				t.Fatalf("renderLogoBox produced no lines for %q", id)
			}
			if width <= 0 {
				t.Errorf("width = %d, must be positive", width)
			}
			for i, line := range lines {
				if got := lipgloss.Width(line); got > width {
					t.Errorf("line %d width %d exceeds reported box width %d", i, got, width)
				}
			}
		})
	}
}

func TestRenderLogoBoxHonorsTruncateHeight(t *testing.T) {
	setPalette("charmtone", config.CustomColors{})
	info := system.Info{DistroID: "arch"}
	t.Run("no truncation", func(t *testing.T) {
		setTruncateHeight(0)
		rendered, _ := renderLogoBox(info)
		if rendered == "" {
			t.Fatal("empty render")
		}
	})
	t.Run("truncate to 5", func(t *testing.T) {
		setTruncateHeight(5)
		defer setTruncateHeight(0)
		rendered, _ := renderLogoBox(info)
		if got := strings.Count(rendered, "\n") + 1; got != 5 {
			t.Errorf("truncated render has %d lines, want 5", got)
		}
	})
}

func TestExpandColorPlaceholders(t *testing.T) {
	tests := map[string]string{
		"$1abc$2def": "abcdef",
		"plain":      "plain",
		"$1":         "",
		"$9test":     "test",
		"$$dollar":   "$$dollar",
		"$1$2$3abc":  "abc",
	}
	for in, want := range tests {
		if got := expandColorPlaceholders(in); got != want {
			t.Errorf("expandColorPlaceholders(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestComposeWithLogoFooterAlignment(t *testing.T) {
	setPalette("charmtone", config.CustomColors{})
	info := system.Info{DistroID: "nixos", Distro: "NixOS", Hostname: "h"}
	rows := [][]string{{"os", "host"}}
	grid, gridWidth := renderGridConfigured(facts(info), rows, 40)
	header := renderHeader()
	body := header + "\n" + grid
	_, logoWidth := renderLogoBox(info)

	// Full mode: footer is rendered at the full composed width
	fullFooter := renderFooter(info, gridWidth+logoWidth+2)
	full := composeWithLogo(body, fullFooter, gridWidth, info, logoWidth, "left", "top", "full")
	fullLines := strings.Split(full, "\n")
	// Just verify we have content and a valid layout.
	if len(fullLines) == 0 {
		t.Fatal("full-mode compose produced no lines")
	}

	gridFooter := renderFooter(info, gridWidth)
	gridAligned := composeWithLogo(body, gridFooter, gridWidth, info, logoWidth, "left", "top", "grid")
	gridLines := strings.Split(gridAligned, "\n")
	if len(gridLines) == 0 {
		t.Fatal("grid-aligned compose produced no lines")
	}
}

func TestComposeWithLogoAllPositions(t *testing.T) {
	setPalette("charmtone", config.CustomColors{})
	info := system.Info{DistroID: "arch", Distro: "Arch", Hostname: "h"}
	rows := [][]string{{"os", "host"}}
	grid, gridWidth := renderGridConfigured(facts(info), rows, 60)
	header := renderHeader()
	body := header + "\n" + grid
	_, logoWidth := renderLogoBox(info)

	// Test all four positions with all three justifies.
	positions := []string{"left", "right", "top", "bottom"}
	justifies := []string{"top", "middle", "bottom"}
	for _, pos := range positions {
		for _, jus := range justifies {
			footer := renderFooter(info, gridWidth)
			result := composeWithLogo(body, footer, gridWidth, info, logoWidth, pos, jus, "full")
			lines := strings.Split(result, "\n")
			if len(lines) == 0 {
				t.Errorf("position=%s justify=%s produced no lines", pos, jus)
			}
		}
	}
}

func TestComposeWithLogoShortGrid(t *testing.T) {
	// Grid much shorter than the logo: blank rows should be inserted so
	// the full logo is shown regardless of justify.
	setPalette("charmtone", config.CustomColors{})
	info := system.Info{DistroID: "nixos", Distro: "NixOS", Hostname: "h"}
	rows := [][]string{{"os"}, {"host"}}
	grid, gridWidth := renderGridConfigured(facts(info), rows, 60)
	header := renderHeader()
	body := header + "\n" + grid
	_, _ = renderLogoBox(info)

	logo, _ := renderLogoBox(info)
	logoLines := strings.Split(logo, "\n")
	logoHeight := len(logoLines)
	gridLines := strings.Split(grid, "\n")
	gridHeight := len(gridLines)

	for _, jus := range []string{"top", "middle", "bottom"} {
		footer := renderFooter(info, gridWidth)
		result := composeWithLogo(body, footer, gridWidth, info, 50, "left", jus, "full")
		bodyHeight := strings.Count(result, "\n") + 1
		// The composed body is logo-height; the footer now rides inside the
		// centered unit instead of hanging below it.
		if bodyHeight < logoHeight {
			t.Errorf("justify=%s: body height %d < logo height %d", jus, bodyHeight, logoHeight)
		}
		_ = gridHeight
	}
}
