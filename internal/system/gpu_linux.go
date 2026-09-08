//go:build linux

package system

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func gpuModels(ctx context.Context) (models []string, igpu string) {
	models = sysfsGPUModels()
	if len(models) == 0 {
		models = lspciGPUModels(ctx)
	}
	for _, model := range models {
		if isIntegratedGPU(model) && igpu == "" {
			igpu = model
		}
	}
	return models, igpu
}

func lspciGPUModels(ctx context.Context) (models []string) {
	if output, err := exec.CommandContext(ctx, "lspci", "-nn").Output(); err == nil {
		for line := range strings.SplitSeq(string(output), "\n") {
			if !strings.Contains(strings.ToLower(line), "vga compatible controller") && !strings.Contains(strings.ToLower(line), "3d controller") {
				continue
			}
			_, description, ok := strings.Cut(line, ": ")
			if !ok {
				continue
			}
			description = cleanGPUModel(description)
			if description != "" && !containsModel(models, description) {
				models = append(models, description)
			}
		}
	}
	if len(models) == 0 {
		if model := strings.TrimSpace(readFileOrEmpty("/sys/class/drm/card0/device/uevent")); model != "" {
			models = append(models, model)
		}
	}
	return models
}

func sysfsGPUModels() (models []string) {
	database := loadPCIDatabase()
	entries, err := os.ReadDir("/sys/class/drm")
	if err != nil || len(database) == 0 {
		return nil
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), "-") || !strings.HasPrefix(entry.Name(), "card") {
			continue
		}
		uevent := readKeyValueFileOrEmpty("/sys/class/drm/" + entry.Name() + "/device/uevent")
		vendor, device, ok := strings.Cut(uevent["PCI_ID"], ":")
		if !ok {
			continue
		}
		model, ok := database[strings.ToLower(vendor)+":"+strings.ToLower(device)]
		if ok && !containsModel(models, model) {
			models = append(models, model)
		}
	}
	return models
}

func loadPCIDatabase() map[string]string {
	paths := []string{"/run/current-system/sw/share/pci.ids", "/usr/share/hwdata/pci.ids", "/usr/share/misc/pci.ids", "/usr/share/pci.ids"}
	if lspci, err := exec.LookPath("lspci"); err == nil {
		if resolved, err := filepath.EvalSymlinks(lspci); err == nil {
			paths = append(paths, filepath.Join(filepath.Dir(filepath.Dir(resolved)), "share", "pci.ids"))
		}
	}
	for _, path := range paths {
		if database := parsePCIDatabase(path); len(database) > 0 {
			return database
		}
	}
	return nil
}

func parsePCIDatabase(path string) map[string]string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	result := make(map[string]string)
	vendor := ""
	for line := range strings.SplitSeq(string(data), "\n") {
		if len(line) < 6 || strings.HasPrefix(line, "#") {
			continue
		}
		if line[0] != ' ' && line[0] != '\t' && line[4] == ' ' {
			vendor = strings.ToLower(line[:4])
			continue
		}
		if vendor == "" || line[0] != '\t' || len(line) < 6 || line[5] != ' ' {
			continue
		}
		device := strings.ToLower(line[1:5])
		name := strings.TrimSpace(line[6:])
		if name != "" {
			result[vendor+":"+device] = cleanGPUModel(withGPUVendor(name, pciVendorName(vendor)))
		}
	}
	return result
}

func pciVendorName(vendor string) string {
	switch vendor {
	case "10de":
		return "NVIDIA"
	case "8086":
		return "Intel"
	case "1002":
		return "AMD"
	default:
		return ""
	}
}

func readKeyValueFileOrEmpty(path string) map[string]string {
	values, err := readKeyValueFile(path)
	if err != nil {
		return map[string]string{}
	}
	return values
}

func cleanGPUModel(model string) string {
	vendor := gpuVendor(model)
	model = strings.TrimSpace(strings.SplitN(model, "(", 2)[0])
	groups := strings.Split(model, "[")
	chosen := strings.TrimSpace(groups[0])
	for _, group := range groups[1:] {
		candidate := strings.TrimSpace(strings.SplitN(group, "]", 2)[0])
		if candidate != "" && !strings.Contains(candidate, ":") && !isPCIClass(candidate) {
			chosen = candidate
			break
		}
	}
	model = chosen
	for _, vendor := range []string{"Intel Corporation ", "Advanced Micro Devices, Inc. [AMD/ATI] ", "Advanced Micro Devices, Inc. ", "NVIDIA Corporation "} {
		if strings.HasPrefix(model, vendor) {
			model = strings.TrimSpace(strings.TrimPrefix(model, vendor))
			break
		}
	}
	return withGPUVendor(strings.TrimSpace(model), vendor)
}

func gpuVendor(model string) string {
	model = strings.ToLower(model)
	switch {
	case strings.Contains(model, "nvidia"):
		return "NVIDIA"
	case strings.Contains(model, "advanced micro devices"), strings.Contains(model, "amd/ati"), strings.Contains(model, "amd"):
		return "AMD"
	case strings.Contains(model, "intel"):
		return "Intel"
	default:
		return ""
	}
}

func withGPUVendor(model, vendor string) string {
	model = strings.TrimSpace(model)
	if model == "" || vendor == "" || strings.HasPrefix(strings.ToLower(model), strings.ToLower(vendor)+" ") {
		return model
	}
	return vendor + " " + model
}

func isPCIClass(value string) bool {
	if len(value) != 4 {
		return false
	}
	for _, r := range value {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') && (r < 'A' || r > 'F') {
			return false
		}
	}
	return true
}

func discreteGPUModels(models []string, igpu string) []string {
	discrete := make([]string, 0, len(models))
	for _, model := range models {
		if model != igpu && !isIntegratedGPU(model) {
			discrete = append(discrete, model)
		}
	}
	return discrete
}

func containsModel(models []string, model string) bool {
	for _, existing := range models {
		if existing == model || strings.Contains(strings.ToLower(existing), strings.ToLower(model)) {
			return true
		}
	}
	return false
}

func isIntegratedGPU(model string) bool {
	model = strings.ToLower(model)
	return strings.Contains(model, "integrated") || strings.Contains(model, "uhd") || strings.Contains(model, "iris") || strings.Contains(model, "vega") || strings.Contains(model, "apu")
}
