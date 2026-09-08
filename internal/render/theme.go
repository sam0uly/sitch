package render

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/exp/charmtone"
	"samouly.fun/sitch/internal/config"
)

var (
	sitchStyle   lipgloss.Style
	headerStyle  lipgloss.Style
	mutedStyle   lipgloss.Style
	valueStyle   lipgloss.Style
	borderStyle  lipgloss.Style
	surfaceStyle lipgloss.Style
	// hasSurfaceBackground reports whether the active palette paints a
	// background surface. With a transparent background we skip injecting
	// blank background padding so unstyled gaps do not stretch or break
	// borders and junctions.
	hasSurfaceBackground bool
	titleColors          []color.Color
	borderColor          color.Color

	charmtoneTitles = []color.Color{
		charmtone.Tang, charmtone.Cumin, charmtone.Paprika, charmtone.Turtle,
		charmtone.Malibu, charmtone.Dolly, charmtone.Violet, charmtone.Sapphire,
		charmtone.Cherry, charmtone.Bengal, charmtone.Mustard, charmtone.Julep,
		charmtone.Guac, charmtone.Zest, charmtone.Damson, charmtone.Oceania,
		charmtone.Mochi,
	}
)

var ttyTitles = []color.Color{
	lipgloss.Color("9"), lipgloss.Color("10"), lipgloss.Color("11"), lipgloss.Color("12"),
	lipgloss.Color("13"), lipgloss.Color("14"), lipgloss.Color("15"), lipgloss.Color("1"),
	lipgloss.Color("2"), lipgloss.Color("3"), lipgloss.Color("4"), lipgloss.Color("5"),
	lipgloss.Color("6"), lipgloss.Color("7"), lipgloss.Color("8"), lipgloss.Color("16"),
}

func init() {
	setPalette("charmtone", config.CustomColors{})
}

func setPalette(mode string, custom config.CustomColors) {
	background, header, value, sitch, border := color.Color(charmtone.Pepper), color.Color(charmtone.Smoke), color.Color(charmtone.Salt), color.Color(charmtone.Charple), color.Color(charmtone.Iron)
	titles := charmtoneTitles
	if mode == "tty" {
		background, header, value, sitch, border = lipgloss.Color("0"), lipgloss.Color("7"), lipgloss.Color("15"), lipgloss.Color("5"), lipgloss.Color("7")
		titles = ttyTitles
	}
	if mode == "custom" {
		background, header, value, sitch, border = transparentColor(custom.Background), transparentColor(custom.Header), transparentColor(custom.Value), transparentColor(custom.Sitch), transparentColor(custom.Border)
		titles = make([]color.Color, 0, len(factOrder))
		for _, spec := range factOrder {
			titles = append(titles, transparentColor(custom.Titles[spec]))
		}
	}
	sitchStyle = lipgloss.NewStyle().Bold(true).Padding(0, 1)
	if mode == "tty" {
		sitchStyle = sitchStyle.Foreground(value).Background(lipgloss.Color("4"))
	} else if empty, _ := isTransparentColor(sitch); empty {
		sitchStyle = sitchStyle.Foreground(value)
	} else {
		sitchStyle = sitchStyle.Foreground(value).Background(sitch)
	}
	headerStyle = maybeForeground(lipgloss.NewStyle(), header)
	mutedStyle = maybeForeground(lipgloss.NewStyle(), header)
	valueStyle = maybeForeground(lipgloss.NewStyle(), value)
	borderStyle = maybeForeground(lipgloss.NewStyle(), border)
	surfaceStyle = maybeBackground(lipgloss.NewStyle(), background)
	titleColors = titles
	borderColor = border
	_, hasSurfaceBackground = background.(lipgloss.NoColor)
	hasSurfaceBackground = !hasSurfaceBackground
}

// transparentColor converts an empty custom palette entry to NoColor, which
// leaves terminal colors alone. It never errors because config validation
// already checked values before this is called. The literal "none" is also
// accepted for explicit transparency.
func transparentColor(value string) color.Color {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || strings.EqualFold(trimmed, "none") {
		return lipgloss.NoColor{}
	}
	return lipgloss.Color(trimmed)
}

// isTransparentColor reports whether a palette entry should be styled with
// NoColor rather than an explicit color.
func isTransparentColor(entry color.Color) (bool, bool) {
	_, isNoColor := entry.(lipgloss.NoColor)
	return isNoColor, true
}

func maybeForeground(style lipgloss.Style, entry color.Color) lipgloss.Style {
	// An explicit NoColor foreground must not leave behind the inherited
	// property from a previous setPalette call.
	if transparent, _ := isTransparentColor(entry); transparent {
		return style.Foreground(lipgloss.NoColor{}).UnsetBackground()
	}
	return style.Foreground(entry).UnsetBackground()
}

func maybeBackground(style lipgloss.Style, entry color.Color) lipgloss.Style {
	if transparent, _ := isTransparentColor(entry); transparent {
		return style
	}
	return style.Background(entry)
}

// styleForeground applies color c as a label foreground, skipping styling
// for transparent custom entries.
func styleForeground(c color.Color) lipgloss.Style {
	if transparent, _ := isTransparentColor(c); transparent {
		return lipgloss.NewStyle().Bold(true)
	}
	return lipgloss.NewStyle().Bold(true).Foreground(c)
}
