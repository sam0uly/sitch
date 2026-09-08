//go:build linux

package system

import "testing"

func TestParseMeminfo(t *testing.T) {
	raw := "MemTotal:       8192000 kB\nMemFree:        1000000 kB\nMemAvailable:   2048000 kB\n"
	used, total, unit := parseMeminfo(raw)
	if used != 6000 || total != 8000 || unit != "MiB" {
		t.Fatalf("memory() = (%d, %d, %q), want (6000, 8000, %q)", used, total, unit, "MiB")
	}
}
