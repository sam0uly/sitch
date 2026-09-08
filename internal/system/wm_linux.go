//go:build linux

package system

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// compositorName returns the process owning the active Wayland socket.
func compositorName() string {
	if display := os.Getenv("WAYLAND_DISPLAY"); display != "" {
		if name := waylandSocketOwner(display); name != "" {
			return name
		}
	}
	return ""
}

func waylandSocketOwner(display string) string {
	if !filepath.IsAbs(display) {
		runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
		if runtimeDir == "" {
			runtimeDir = "/run/user/" + strconv.Itoa(os.Getuid())
		}
		display = filepath.Join(runtimeDir, display)
	}
	socketInode := waylandSocketInode(display)
	if socketInode == "" {
		return ""
	}
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return ""
	}
	needle := "socket:[" + socketInode + "]"
	for _, entry := range entries {
		if _, err := strconv.Atoi(entry.Name()); err != nil {
			continue
		}
		fds, err := os.ReadDir("/proc/" + entry.Name() + "/fd")
		if err != nil {
			continue
		}
		for _, fd := range fds {
			link, err := os.Readlink("/proc/" + entry.Name() + "/fd/" + fd.Name())
			if err == nil && link == needle {
				if commandLine, err := os.ReadFile("/proc/" + entry.Name() + "/cmdline"); err == nil {
					if command := strings.Split(string(commandLine), "\x00")[0]; command != "" {
						return filepath.Base(command)
					}
				}
				executable, err := os.Readlink("/proc/" + entry.Name() + "/exe")
				if err == nil {
					return filepath.Base(executable)
				}
			}
		}
	}
	return ""
}

func waylandSocketInode(display string) string {
	data, err := os.ReadFile("/proc/net/unix")
	if err != nil {
		return ""
	}
	for line := range strings.SplitSeq(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 8 || fields[len(fields)-1] != display || fields[5] != "01" {
			continue
		}
		return fields[6]
	}
	return ""
}
