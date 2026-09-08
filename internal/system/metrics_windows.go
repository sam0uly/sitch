//go:build windows

package system

import (
	"os"
	"strconv"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var procGlobalMemoryStatusEx = windows.NewLazySystemDLL("kernel32.dll").NewProc("GlobalMemoryStatusEx")

type memoryStatusEx struct {
	Length               uint32
	DwMemoryLoad         uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

func memory() (used, total uint64, unit string) {
	var m memoryStatusEx
	m.Length = uint32(unsafe.Sizeof(m))
	r1, _, _ := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&m)))
	if r1 == 0 || m.TotalPhys == 0 {
		return 0, 0, ""
	}
	return (m.TotalPhys - m.AvailPhys) / (1024 * 1024), m.TotalPhys / (1024 * 1024), "MiB"
}

func diskUsage(path string) (used, total uint64) {
	ptr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, 0
	}
	var avail, totalBytes, freeBytes uint64
	if err := windows.GetDiskFreeSpaceEx(ptr, &avail, &totalBytes, &freeBytes); err != nil {
		return 0, 0
	}
	if avail > totalBytes {
		return 0, totalBytes
	}
	return totalBytes - avail, totalBytes
}

func osAge(path string) string {
	fi, err := os.Stat(path)
	if err != nil {
		return "unknown"
	}
	data, ok := fi.Sys().(*syscall.Win32FileAttributeData)
	if !ok {
		return "unknown"
	}
	installed := time.Unix(0, data.CreationTime.Nanoseconds())
	if installed.IsZero() || installed.After(time.Now()) {
		return "unknown"
	}
	days := int(time.Since(installed) / (24 * time.Hour))
	return strconv.Itoa(days) + " days"
}
