package render

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"samouly.fun/sitch/internal/system"
)

func TestConfiguredRowsKeepDynamicBordersAligned(t *testing.T) {
	items := facts(system.Info{
		Distro: "OS", HostModel: "HOST", Kernel: "KERNEL", Uptime: "UPTIME",
		GPU: "GPU", IGPU: "IGPU", CPU: "CPU", Packages: "PACKAGES",
		Shell: "SHELL", Display: "DISPLAY", Desktop: "DESKTOP", Terminal: "TERMINAL",
		OSAge: "AGE", Locale: "LOCALE",
	})
	for pattern := 0; pattern < 729; pattern++ {
		rows := make([][]string, 0, 6)
		shape := pattern
		for i := 0; i < 6; i++ {
			columns := 1 + shape%3
			shape /= 3
			row := make([]string, columns)
			for column := range row {
				row[column] = items[(i*3+column)%len(items)].key
			}
			rows = append(rows, row)
		}
		grid, _ := renderGridConfigured(items, rows, 160)
		lines := strings.Split(grid, "\n")
		width := lipgloss.Width(lines[0])
		for _, line := range lines {
			if got := lipgloss.Width(line); got != width {
				t.Fatalf("pattern %d: line width = %d, want %d\n%s", pattern, got, width, grid)
			}
		}
		plain := ansi.Strip(grid)
		if !strings.Contains(plain, "└") || !strings.Contains(plain, "┘") {
			t.Fatalf("pattern %d: missing bottom corners\n%s", pattern, plain)
		}
	}
}
