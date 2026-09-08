//go:build !linux && !darwin && !windows

package system

func memory() (used, total uint64, unit string) { return 0, 0, "" }

func diskUsage(string) (used, total uint64) { return 0, 0 }

func osAge(string) string { return "unknown" }
