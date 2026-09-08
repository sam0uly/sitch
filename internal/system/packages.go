package system

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

func packageCounts(ctx context.Context, distro string) (system, user, total int) {
	var count int
	switch distro {
	case "arch", "artix", "archcraft", "manjaro", "endeavouros", "garuda":
		count = directoryCount("/var/lib/pacman/local")
	case "debian", "ubuntu", "pop", "linuxmint":
		count = prefixedLineCount("/var/lib/dpkg/status", "Package:")
	case "nixos":
		home := os.Getenv("HOME")
		stateHome := os.Getenv("XDG_STATE_HOME")
		if stateHome == "" {
			stateHome = home + "/.local/state"
		}
		counts := nixStorePackageCounts(
			ctx,
			"/run/current-system",
			home+"/.nix-profile",
			stateHome+"/nix/profile",
			"/etc/profiles/per-user/"+firstNonEmpty(os.Getenv("USER"), os.Getenv("LOGNAME")),
		)
		return counts[0], counts[1] + counts[2] + counts[3], 0
	case "gentoo":
		count = packageDirectoryCount("/var/db/pkg")
	case "fedora", "rhel", "centos", "redhat", "opensuse":
		count = commandLineCount(ctx, "rpm", "-qa")
	case "void":
		count = commandLineCount(ctx, "xbps-query", "-l")
	}
	return 0, 0, count
}

func nixStorePackageCounts(ctx context.Context, profiles ...string) []int {
	if cached, ok := readNixPackageCache(profiles); ok {
		return cached
	}
	counts := queryNixStorePackageCounts(ctx, profiles...)
	if counts[0] > 0 || counts[1] > 0 || counts[2] > 0 || counts[3] > 0 {
		writeNixPackageCache(profiles, counts[0], counts[1]+counts[2]+counts[3])
	}
	return counts
}

func queryNixStorePackageCounts(ctx context.Context, profiles ...string) []int {
	counts := make([]int, len(profiles))
	var wait sync.WaitGroup
	for i, profile := range profiles {
		wait.Add(1)
		go func() {
			defer wait.Done()
			counts[i] = nixStorePackageCount(ctx, profile)
		}()
	}
	wait.Wait()
	return counts
}

type nixPackageCache struct {
	Key    string `json:"key"`
	System int    `json:"system"`
	User   int    `json:"user"`
}

func nixPackageCacheKey(profiles []string) string {
	parts := make([]string, len(profiles))
	for i, profile := range profiles {
		resolved, err := filepath.EvalSymlinks(profile)
		if err != nil {
			resolved = "missing"
		}
		parts[i] = profile + "=" + resolved
	}
	return strings.Join(parts, "|")
}

func nixPackageCachePath() string {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(cacheDir, "sitch", "nix-packages.json")
}

func readNixPackageCache(profiles []string) ([]int, bool) {
	path := nixPackageCachePath()
	if path == "" {
		return nil, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var cache nixPackageCache
	if json.Unmarshal(data, &cache) != nil || cache.Key != nixPackageCacheKey(profiles) {
		return nil, false
	}
	return []int{cache.System, cache.User, 0, 0}, true
}

func writeNixPackageCache(profiles []string, system, user int) {
	path := nixPackageCachePath()
	if path == "" {
		return
	}
	data, err := json.Marshal(nixPackageCache{Key: nixPackageCacheKey(profiles), System: system, User: user})
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return
	}
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, data, 0o600); err == nil {
		_ = os.Rename(temporary, path)
	}
}

func nixStorePackageCount(ctx context.Context, profile string) int {
	if _, err := os.Stat(profile); err != nil {
		return 0
	}
	output, err := exec.CommandContext(ctx, "nix-store", "--query", "--requisites", profile).Output()
	if err != nil {
		return 0
	}
	count := 0
	for line := range strings.SplitSeq(string(output), "\n") {
		if isNixPackagePath(line) {
			count++
		}
	}
	return count
}

func isNixPackagePath(path string) bool {
	if path == "" {
		return false
	}
	name := filepath.Base(path)
	parts := strings.SplitN(name, "-", 2)
	if len(parts) != 2 || len(parts[0]) != 32 {
		return false
	}
	name = parts[1]
	for _, suffix := range []string{"-doc", "-man", "-info", "-dev", "-bin"} {
		if strings.HasSuffix(name, suffix) {
			return false
		}
	}
	if strings.HasPrefix(name, "nixos-system-nixos-") {
		return false
	}
	for i := 0; i+2 < len(name); i++ {
		if name[i] >= '0' && name[i] <= '9' && name[i+1] == '.' && name[i+2] >= '0' && name[i+2] <= '9' {
			return true
		}
	}
	return false
}

func directoryCount(path string) int {
	entries, err := os.ReadDir(path)
	if err != nil {
		return 0
	}
	return len(entries)
}

func prefixedLineCount(path, prefix string) int {
	count := 0
	for line := range strings.SplitSeq(readFileOrEmpty(path), "\n") {
		if strings.HasPrefix(line, prefix) {
			count++
		}
	}
	return count
}

func packageDirectoryCount(root string) int {
	distros, err := os.ReadDir(root)
	if err != nil {
		return 0
	}
	count := 0
	for _, distro := range distros {
		if distro.IsDir() {
			count += directoryCount(root + "/" + distro.Name())
		}
	}
	return count
}

func commandLineCount(ctx context.Context, name string, args ...string) int {
	output, err := exec.CommandContext(ctx, name, args...).Output()
	if err != nil {
		return 0
	}
	output = []byte(strings.TrimSpace(string(output)))
	if len(output) == 0 {
		return 0
	}
	return len(strings.Split(string(output), "\n"))
}
