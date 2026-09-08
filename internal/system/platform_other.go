//go:build !linux

package system

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func collectPlatform(parent context.Context) (Info, error) {
	ctx, cancel := context.WithTimeout(parent, 1500*time.Millisecond)
	defer cancel()
	hostname, _ := os.Hostname()
	info := Info{
		User:      firstNonEmpty(os.Getenv("USER"), os.Getenv("USERNAME")),
		Hostname:  firstNonEmpty(hostname, os.Getenv("HOSTNAME"), os.Getenv("COMPUTERNAME")),
		DistroID:  runtime.GOOS,
		Distro:    runtime.GOOS,
		Kernel:    firstNonEmpty(commandOutput(ctx, "uname", "-sr"), "unknown"),
		Shell:     baseName(firstNonEmpty(os.Getenv("SHELL"), os.Getenv("ComSpec"))),
		Terminal:  firstNonEmpty(os.Getenv("TERM_PROGRAM"), os.Getenv("TERM")),
		Locale:    firstNonEmpty(os.Getenv("LANG"), os.Getenv("LC_ALL")),
		Packages:  "unknown",
		HostModel: "unknown",
		CPU:       "unknown",
		GPU:       "unknown",
		Desktop:   "unknown",
		Display:   "unknown",
		OSAge:     "unknown",
	}
	info.MemoryUsed, info.MemoryTotal, info.MemoryUnit = memory()
	info.DiskUsed, info.DiskTotal = diskUsage("/")
	info.OSAge = osAge(osAgePath())
	if runtime.GOOS == "darwin" {
		info.HostModel = firstNonEmpty(commandOutput(ctx, "sysctl", "-n", "hw.model"), "unknown")
		info.CPU = firstNonEmpty(commandOutput(ctx, "sysctl", "-n", "machdep.cpu.brand_string"), "unknown")
		info.Distro = firstNonEmpty(commandOutput(ctx, "sw_vers", "-productName"), "macOS")
		info.Kernel = firstNonEmpty(commandOutput(ctx, "uname", "-sr"), "unknown")
	}
	if runtime.GOOS == "windows" {
		info.HostModel = firstNonEmpty(os.Getenv("COMPUTERNAME"), "unknown")
		info.Kernel = firstNonEmpty(os.Getenv("OS"), "Windows")
		info.CPU = firstNonEmpty(os.Getenv("PROCESSOR_IDENTIFIER"), "unknown")
	}
	return info, nil
}

func commandOutput(ctx context.Context, name string, args ...string) string {
	output, err := exec.CommandContext(ctx, name, args...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func osAgePath() string {
	if runtime.GOOS == "windows" {
		drive := os.Getenv("SystemDrive")
		if drive == "" {
			drive = "C:"
		}
		return filepath.Join(drive, "Windows")
	}
	return "/"
}
