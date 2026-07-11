package platform

import (
	"bufio"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"perftool/internal/model"
)

func Host() model.HostInfo {
	host := model.HostInfo{
		OS: runtime.GOOS, Architecture: runtime.GOARCH,
		LogicalCPUs: runtime.NumCPU(),
	}
	if data, err := os.ReadFile("/proc/sys/kernel/osrelease"); err == nil {
		host.Kernel = strings.TrimSpace(string(data))
	}
	host.CPU = cpuName()
	host.MemoryTotalBytes, host.MemoryBeforeBytes = Memory()
	host.ProcessesBefore = countProcesses()
	if data, err := os.ReadFile("/proc/loadavg"); err == nil {
		fields := strings.Fields(string(data))
		if len(fields) >= 3 {
			host.LoadBefore = strings.Join(fields[:3], " ")
		}
	}
	return host
}

func cpuName() string {
	file, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return ""
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		key, value, ok := strings.Cut(scanner.Text(), ":")
		if ok && strings.TrimSpace(key) == "model name" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func Memory() (uint64, uint64) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0
	}
	defer file.Close()
	var total, available uint64
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		value, _ := strconv.ParseUint(fields[1], 10, 64)
		switch strings.TrimSuffix(fields[0], ":") {
		case "MemTotal":
			total = value * 1024
		case "MemAvailable":
			available = value * 1024
		}
	}
	return total, available
}

func countProcesses() int {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return 0
	}
	count := 0
	for _, entry := range entries {
		if _, err := strconv.Atoi(entry.Name()); err == nil && entry.IsDir() {
			count++
		}
	}
	return count
}

func ResourceUsage(state *os.ProcessState) (float64, float64, uint64) {
	var maxRSS uint64
	if usage, ok := state.SysUsage().(*syscall.Rusage); ok && usage.Maxrss > 0 {
		maxRSS = uint64(usage.Maxrss) * 1024
	}
	return state.UserTime().Seconds(), state.SystemTime().Seconds(), maxRSS
}
