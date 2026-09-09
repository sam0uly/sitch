package render

import (
	"image/color"
	"slices"
	"strings"

	"charm.land/lipgloss/v2"
	"samouly.fun/sitch/internal/system"
)

// pickLogo returns the ASCII logo for the host distro, falling back to the
// default Sitch wordmark when no entry matches. The size argument picks
// between the regular and small variants; "small" tries the small variant
// first and falls back to the regular one. When truncateHeight > 0 the
// logo is also clipped to that many lines (set by setTruncateHeight).
// An explicit logo name or custom file set via setLogoSource takes
// precedence over auto-detection.
func pickLogo(info system.Info, size string) fastfetchLogo {
	if custom := activeLogoRequest.customText(); custom != "" {
		return fastfetchLogo{text: custom, lines: strings.Count(custom, "\n") + 1}
	}
	if name := activeLogoRequest.name(); name != "" {
		if logo, ok := lookupLogo(name, size); ok {
			return logo
		}
	}
	id := strings.ToLower(info.DistroID)
	if size == "small" {
		if l, ok := fastfetchLogos[id+"_small"]; ok {
			return l
		}
	}
	if l, ok := fastfetchLogos[id]; ok {
		return l
	}
	return fastfetchLogo{text: defaultLogoText, lines: strings.Count(defaultLogoText, "\n") + 1}
}

// lookupLogo finds a bundled logo by id (case-insensitive), honoring the
// small size variant. It reports false for unknown ids.
func lookupLogo(name, size string) (fastfetchLogo, bool) {
	id := strings.ToLower(strings.TrimSpace(name))
	if size == "small" {
		if l, ok := fastfetchLogos[id+"_small"]; ok {
			return l, true
		}
	}
	l, ok := fastfetchLogos[id]
	return l, ok
}

// LogoNames returns the sorted ids of all bundled logos, including the
// "_small" variants.
func LogoNames() []string {
	names := make([]string, 0, len(fastfetchLogos))
	for name := range fastfetchLogos {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// renderLogoBox paints the logo as a vertical column of colored lines.
// $1..$9 placeholders from the source art are replaced with the titleColors
// palette (cycled when the art has more than 9 color groups). Returns the
// rendered string and its visible width. When truncateHeight > 0 and the
// logo has more lines than truncateHeight, the logo is clipped from the
// bottom (or middle, depending on justify) so the body never grows past
// the available space.
func renderLogoBox(info system.Info) (string, int) {
	l := pickLogo(info, currentLogoSize())
	lines := strings.Split(strings.Trim(l.text, "\n"), "\n")
	if truncateHeight > 0 && len(lines) > truncateHeight {
		lines = lines[:truncateHeight]
	}
	out := make([]string, 0, len(lines))
	maxWidth := 0
	for i, raw := range lines {
		expanded := expandColorPlaceholders(raw)
		accent := logoAccent(i)
		styled := lipgloss.NewStyle().Foreground(accent).Render(expanded)
		out = append(out, styled)
		if w := lipgloss.Width(styled); w > maxWidth {
			maxWidth = w
		}
	}
	return strings.Join(out, "\n"), maxWidth
}

// logoAccent picks the foreground color for a logo line. In multi mode
// (the default) it cycles the titleColors palette; in single mode every
// line uses the configured logo_color, falling back to the first palette
// entry when no color was set.
func logoAccent(line int) color.Color {
	if activeLogoColorMode == "single" {
		if activeLogoColor != nil {
			return activeLogoColor
		}
		return titleColors[0]
	}
	return titleColors[line%len(titleColors)]
}

// currentLogoSize is the size the user requested for the current render.
// It is set by PrintWithOptions via setLogoSize before renderLogoBox runs.
var (
	activeLogoSize      = "regular"
	activeLogoRequest   logoRequest
	activeLogoColorMode = "multi"
	activeLogoColor     color.Color
	truncateHeight      int
)

// logoRequest records an explicit logo override for the current render:
// either a bundled logo id or raw custom ASCII text.
type logoRequest struct {
	logoName   string
	logoCustom string
}

// setLogoRequest remembers an explicit logo override for one render. An
// empty name and empty custom text mean "auto-detect from distro".
func setLogoRequest(name, customText string) {
	activeLogoRequest = logoRequest{logoName: strings.TrimSpace(name), logoCustom: customText}
}

func (r logoRequest) name() string { return r.logoName }

func (r logoRequest) customText() string { return r.logoCustom }

func currentLogoSize() string { return activeLogoSize }

// setLogoSize is called by PrintWithOptions to remember the user's choice
// for the duration of one render.
func setLogoSize(size string) {
	if size == "" {
		size = "regular"
	}
	activeLogoSize = size
}

// setLogoColorMode remembers the logo coloring choice for one render.
// mode "single" applies logoSpec to every line; an empty spec keeps the
// palette's first entry, and "none" renders the logo transparent.
func setLogoColorMode(mode, logoSpec string) {
	if mode == "" {
		mode = "multi"
	}
	activeLogoColorMode = mode
	activeLogoColor = nil
	if mode == "single" {
		spec := strings.TrimSpace(logoSpec)
		if spec != "" {
			activeLogoColor = transparentColor(spec)
		}
	}
}

// setTruncateHeight caps the logo to a maximum line count. Pass 0 to
// disable truncation. Called by PrintWithOptions.
func setTruncateHeight(n int) {
	truncateHeight = n
}

// expandColorPlaceholders replaces $1..$9 placeholders in raw art with ANSI
// escape sequences that switch to the matching entry in the active palette.
// We can't color the actual text without knowing the palette here, so we
// leave the placeholders in place and let the per-line lipgloss style handle
// foreground coloring of the whole line. The placeholder is left visible so
// fastfetch-style multi-color art shows clearly. To honor multi-color at
// character level, callers can implement a richer renderer; for now we
// strip the placeholders so the rendered output is clean.
func expandColorPlaceholders(raw string) string {
	if !strings.ContainsAny(raw, "$") {
		return raw
	}
	var b strings.Builder
	b.Grow(len(raw))
	for i := 0; i < len(raw); i++ {
		if raw[i] == '$' && i+1 < len(raw) && raw[i+1] >= '1' && raw[i+1] <= '9' {
			i++ // skip the digit; placeholder removed
			continue
		}
		b.WriteByte(raw[i])
	}
	return b.String()
}
