// Package system collects host facts for the Sitch system fetch.
package system

import (
	"context"
	"os"
	"strings"
)

// Info is one consistent snapshot of the host.
type Info struct {
	User          string   `json:"user"`
	Hostname      string   `json:"hostname"`
	HostModel     string   `json:"host_model"`
	DistroID      string   `json:"distro_id"`
	Distro        string   `json:"distro"`
	Kernel        string   `json:"kernel"`
	Uptime        string   `json:"uptime"`
	Shell         string   `json:"shell"`
	Packages      string   `json:"packages"`
	PackageSystem int      `json:"package_system"`
	PackageUser   int      `json:"package_user"`
	CPU           string   `json:"cpu"`
	GPU           string   `json:"gpu"`
	GPUs          []string `json:"gpus,omitempty"`
	IGPU          string   `json:"igpu,omitempty"`
	Desktop       string   `json:"desktop"`
	Terminal      string   `json:"terminal"`
	Display       string   `json:"display"`
	Locale        string   `json:"locale"`
	OSAge         string   `json:"os_age"`
	MemoryUsed    uint64   `json:"memory_used"`
	MemoryTotal   uint64   `json:"memory_total"`
	MemoryUnit    string   `json:"memory_unit"`
	DiskUsed      uint64   `json:"disk_used"`
	DiskTotal     uint64   `json:"disk_total"`
}

// Collect reads a Linux host snapshot. Optional facts remain "unknown" when
// the host does not expose the corresponding file or environment variable.
func Collect() (Info, error) {
	return collectPlatform(context.Background())
}

func readFileOrEmpty(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(b)
}

func readKeyValueFile(path string) (map[string]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	values := make(map[string]string, 16)
	for line := range strings.SplitSeq(string(b), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		values[key] = strings.Trim(value, "\"'")
	}
	return values, nil
}

func kernelVersion(versionLine string) string {
	fields := strings.Fields(versionLine)
	if len(fields) >= 3 {
		return fields[2]
	}
	return "unknown"
}

func cpuModel(raw string) string {
	for line := range strings.SplitSeq(raw, "\n") {
		if key, value, ok := strings.Cut(line, ":"); ok && strings.TrimSpace(key) == "model name" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func cleanCPUModel(model string) string {
	model = strings.NewReplacer("(R)", "", "(TM)", "", "(C)", "").Replace(model)
	if index := strings.Index(model, " @"); index >= 0 {
		model = model[:index]
	}
	model = strings.TrimSpace(strings.TrimSuffix(model, " CPU"))
	return strings.Join(strings.Fields(model), " ")
}

func baseName(value string) string {
	if value == "" {
		return "unknown"
	}
	return value[strings.LastIndexByte(value, '/')+1:]
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return "unknown"
}

func desktopEnvironment() string {
	if compositor := compositorName(); compositor != "" {
		return compositor
	}
	return firstNonEmpty(os.Getenv("XDG_SESSION_DESKTOP"), os.Getenv("XDG_CURRENT_DESKTOP"))
}

func displayEnvironment() string {
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		return "Wayland"
	}
	if os.Getenv("DISPLAY") != "" {
		return "X11"
	}
	return firstNonEmpty(os.Getenv("XDG_SESSION_TYPE"), "unknown")
}
