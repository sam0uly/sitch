//go:build darwin

package system

import (
	"context"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

func memory() (used, total uint64, unit string) {
	total, err := unix.SysctlUint64("hw.memsize")
	if err != nil || total == 0 {
		return 0, 0, ""
	}
	pageSize, availablePages := parseVmStat(commandOutput(context.Background(), "vm_stat"))
	if pageSize == 0 {
		pageSize = uint64(os.Getpagesize())
	}
	if availablePages == 0 {
		return 0, 0, ""
	}
	available := min(availablePages*pageSize, total)
	return (total - available) / (1024 * 1024), total / (1024 * 1024), "MiB"
}

func parseVmStat(raw string) (pageSize, availablePages uint64) {
	for line := range strings.SplitSeq(raw, "\n") {
		line = strings.TrimSuffix(strings.TrimSpace(line), ".")
		if strings.HasPrefix(line, "Mach Virtual Memory Statistics: (page size of ") {
			fields := strings.Fields(line)
			if len(fields) >= 8 {
				if v, err := strconv.ParseUint(fields[7], 10, 64); err == nil {
					pageSize = v
				}
			}
			continue
		}
		fields := strings.SplitN(line, ":", 2)
		if len(fields) != 2 {
			continue
		}
		value, err := strconv.ParseUint(strings.TrimSpace(fields[1]), 10, 64)
		if err != nil {
			continue
		}
		switch fields[0] {
		case "Pages free", "Pages inactive":
			availablePages += value
		}
	}
	return
}

func diskUsage(path string) (used, total uint64) {
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return 0, 0
	}
	total = stat.Blocks * uint64(stat.Bsize)
	free := stat.Bavail * uint64(stat.Bsize)
	if free > total {
		return 0, total
	}
	return total - free, total
}

func osAge(path string) string {
	fi, err := os.Stat(path)
	if err != nil {
		return "unknown"
	}
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok || st.Birthtimespec.Sec == 0 {
		return "unknown"
	}
	installed := time.Unix(st.Birthtimespec.Sec, st.Birthtimespec.Nsec)
	if installed.After(time.Now()) {
		return "unknown"
	}
	days := int(time.Since(installed) / (24 * time.Hour))
	return strconv.Itoa(days) + " days"
}
