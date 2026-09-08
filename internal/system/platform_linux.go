//go:build linux

package system

import (
	"context"
	"os"
	"strings"
	"time"
)

func collectPlatform(ctx context.Context) (Info, error) {
	osRelease, err := readKeyValueFile("/etc/os-release")
	if err != nil {
		return Info{}, err
	}
	info := Info{
		User:      firstNonEmpty(os.Getenv("USER"), os.Getenv("LOGNAME")),
		Hostname:  firstNonEmpty(strings.TrimSpace(readFileOrEmpty("/etc/hostname")), os.Getenv("HOSTNAME")),
		DistroID:  osRelease["ID"],
		Distro:    firstNonEmpty(osRelease["PRETTY_NAME"], osRelease["NAME"], osRelease["ID"]),
		Kernel:    firstNonEmpty(strings.TrimSpace(readFileOrEmpty("/proc/sys/kernel/osrelease")), kernelVersion(readFileOrEmpty("/proc/version"))),
		Uptime:    formatUptime(readFileOrEmpty("/proc/uptime")),
		Shell:     baseName(os.Getenv("SHELL")),
		HostModel: firstNonEmpty(strings.TrimSpace(readFileOrEmpty("/sys/devices/virtual/dmi/id/product_name")), "unknown"),
		CPU:       firstNonEmpty(cleanCPUModel(cpuModel(readFileOrEmpty("/proc/cpuinfo"))), "unknown"),
		Desktop:   desktopEnvironment(),
		Terminal:  firstNonEmpty(os.Getenv("TERM_PROGRAM"), os.Getenv("TERM")),
		Display:   displayEnvironment(),
		Locale:    firstNonEmpty(os.Getenv("LANG"), os.Getenv("LC_ALL"), os.Getenv("LC_CTYPE")),
		OSAge:     osAge("/"),
	}
	hardwareCtx, hardwareCancel := context.WithTimeout(ctx, 2*time.Second)
	info.GPUs, info.IGPU = gpuModels(hardwareCtx)
	hardwareCancel()
	info.GPU = firstNonEmpty(strings.Join(discreteGPUModels(info.GPUs, info.IGPU), ", "), "unknown")
	info.MemoryUsed, info.MemoryTotal, info.MemoryUnit = memory()
	info.DiskUsed, info.DiskTotal = diskUsage("/")
	packageCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	info.PackageSystem, info.PackageUser, _ = packageCounts(packageCtx, info.DistroID)
	return info, nil
}
