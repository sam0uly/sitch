//go:build !linux

package system

func memory(string) (used, total uint64, unit string) { return 0, 0, "" }

func diskUsage(string) (used, total uint64) { return 0, 0 }

func osAge(string) string { return "unknown" }
