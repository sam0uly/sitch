//go:build darwin

package system

import "testing"

func TestParseVmStat(t *testing.T) {
	raw := "Mach Virtual Memory Statistics: (page size of 4096 bytes)\n" +
		"Pages free:                              12345.\n" +
		"Pages active:                           987654.\n" +
		"Pages inactive:                         111111.\n" +
		"Pages wired down:                       222222.\n"
	pageSize, available := parseVmStat(raw)
	if pageSize != 4096 || available != 12345+111111 {
		t.Fatalf("parseVmStat() = (%d, %d), want (4096, %d)", pageSize, available, 12345+111111)
	}
}
