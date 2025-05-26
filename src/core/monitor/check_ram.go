package monitor

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

type RamStats struct {
	TotalGB     float64
	AvailableGB float64
	UsedPercent float64
}

func GetRamStats() (*RamStats, error) {
	switch runtime.GOOS {
	case "windows":
		return getRamStatsWindows()
	case "linux":
		return getRamStatsLinux()
	case "darwin":
		return getRamStatsMacOS()
	default:
		return nil, fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

type memStatusEx struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

func getRamStatsWindows() (*RamStats, error) {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	proc := kernel32.NewProc("GlobalMemoryStatusEx")

	var mem memStatusEx
	mem.Length = uint32(unsafe.Sizeof(mem))
	ret, _, err := proc.Call(uintptr(unsafe.Pointer(&mem)))
	if ret == 0 {
		return nil, fmt.Errorf("GlobalMemoryStatusEx failed: %w", err)
	}

	total := float64(mem.TotalPhys) / (1024 * 1024 * 1024)
	avail := float64(mem.AvailPhys) / (1024 * 1024 * 1024)

	return &RamStats{
		TotalGB:     total,
		AvailableGB: avail,
		UsedPercent: (total - avail) * 100 / total,
	}, nil
}

func getRamStatsLinux() (*RamStats, error) {
	content, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return nil, err
	}

	memInfo := make(map[string]uint64)
	for _, line := range strings.Split(string(content), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		key := strings.TrimSuffix(fields[0], ":")
		if value, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
			memInfo[key] = value
		}
	}

	total := float64(memInfo["MemTotal"]) / (1024 * 1024)
	available := float64(memInfo["MemAvailable"]) / (1024 * 1024)
	if available == 0 {
		available = float64(memInfo["MemFree"]) / (1024 * 1024)
	}

	return &RamStats{
		TotalGB:     total,
		AvailableGB: available,
		UsedPercent: (total - available) * 100 / total,
	}, nil
}

func getRamStatsMacOS() (*RamStats, error) {
	cmd := exec.Command("sysctl", "-n", "hw.memsize")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get total memory: %w", err)
	}

	totalBytes, err := strconv.ParseUint(strings.TrimSpace(string(output)), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse total memory: %w", err)
	}

	cmd = exec.Command("vm_stat")
	output, err = cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get vm_stat: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	pageSize := uint64(4096)
	freePages := uint64(0)
	inactivePages := uint64(0)

	for _, line := range lines {
		if strings.Contains(line, "page size of") {
			parts := strings.Split(line, "page size of ")
			if len(parts) > 1 {
				size, err := strconv.ParseUint(strings.TrimSpace(strings.Split(parts[1], " ")[0]), 10, 64)
				if err == nil {
					pageSize = size
				}
			}
		} else if strings.Contains(line, "Pages free:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				free, err := strconv.ParseUint(strings.TrimSpace(strings.ReplaceAll(parts[1], ".", "")), 10, 64)
				if err == nil {
					freePages = free
				}
			}
		} else if strings.Contains(line, "Pages inactive:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				inactive, err := strconv.ParseUint(strings.TrimSpace(strings.ReplaceAll(parts[1], ".", "")), 10, 64)
				if err == nil {
					inactivePages = inactive
				}
			}
		}
	}

	totalGB := float64(totalBytes) / (1024 * 1024 * 1024)
	availableBytes := (freePages + inactivePages) * pageSize
	availableGB := float64(availableBytes) / (1024 * 1024 * 1024)

	if availableGB <= 0 || availableGB > totalGB {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		availableGB = totalGB - float64(m.Sys)/(1024*1024*1024)
	}

	return &RamStats{
		TotalGB:     totalGB,
		AvailableGB: availableGB,
		UsedPercent: (totalGB - availableGB) * 100 / totalGB,
	}, nil
}
