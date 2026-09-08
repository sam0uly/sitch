package render

import (
	"fmt"
	"image/color"

	"samouly.fun/sitch/internal/system"
)

type fact struct {
	key   string
	icon  string
	label string
	value string
}

type gridCell struct {
	item   fact
	accent color.Color
	width  int
}

var factOrder = []string{
	"os", "host", "disk", "memory", "kernel", "uptime", "gpu", "igpu",
	"cpu", "packages", "shell", "display", "desktop", "terminal", "os-age", "locale",
}

func facts(info system.Info) []fact {
	return []fact{
		{key: "os", icon: "󰀄", label: "OS", value: info.Distro},
		{key: "host", icon: "󰌢", label: "HOST", value: info.HostModel},
		{key: "disk", icon: "󰋊", label: "DISK", value: diskText(info)},
		{key: "memory", icon: "󰑭", label: "MEMORY", value: memoryText(info)},
		{key: "kernel", icon: "󰍛", label: "KERNEL", value: info.Kernel},
		{key: "uptime", icon: "󰅐", label: "UPTIME", value: info.Uptime},
		{key: "gpu", icon: "󰢮", label: "GPU", value: info.GPU},
		{key: "igpu", icon: "󰘚", label: "IGPU", value: first(info.IGPU)},
		{key: "cpu", icon: "󰍛", label: "CPU", value: info.CPU},
		{key: "packages", icon: "󰏖", label: "PACKAGES", value: packagesText(info)},
		{key: "shell", icon: "󰈔", label: "SHELL", value: info.Shell},
		{key: "display", icon: "󰖲", label: "DISPLAY", value: info.Display},
		{key: "desktop", icon: "󰇄", label: "DESKTOP", value: info.Desktop},
		{key: "terminal", icon: "󰆍", label: "TERMINAL", value: info.Terminal},
		{key: "os-age", icon: "󰅀", label: "OS AGE", value: info.OSAge},
		{key: "locale", icon: "󰗚", label: "LOCALE", value: info.Locale},
	}
}

func first(value string) string {
	if value == "" {
		return "unknown"
	}
	return value
}

func packagesText(info system.Info) string {
	system, user := info.PackageSystem, info.PackageUser
	if system == 0 && user == 0 {
		return "unknown"
	}
	if user == 0 {
		return fmt.Sprintf("%d system", system)
	}
	if system == 0 {
		return fmt.Sprintf("%d user", user)
	}
	return fmt.Sprintf("%d system / %d user", system, user)
}
