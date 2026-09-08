//go:build linux

package system

import (
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

func memory() (used, total uint64, unit string) {
	return parseMeminfo(readFileOrEmpty("/proc/meminfo"))
}

func parseMeminfo(raw string) (used, total uint64, unit string) {
	var available uint64
	for line := range strings.SplitSeq(raw, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}
		switch strings.TrimSuffix(fields[0], ":") {
		case "MemTotal":
			total = value / 1024
		case "MemAvailable":
			available = value / 1024
		}
	}
	if available > total {
		available = total
	}
	return total - available, total, "MiB"
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
	var stat unix.Statx_t
	if err := unix.Statx(unix.AT_FDCWD, path, unix.AT_STATX_SYNC_AS_STAT, unix.STATX_BTIME, &stat); err != nil || stat.Btime.Sec == 0 {
		return "unknown"
	}
	installed := time.Unix(stat.Btime.Sec, int64(stat.Btime.Nsec))
	if installed.After(time.Now()) {
		return "unknown"
	}
	days := int(time.Since(installed) / (24 * time.Hour))
	return strconv.Itoa(days) + " days"
}
