package platform

import (
	"bufio"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"perftool/internal/model"
)

func Host() model.HostInfo {
	host := model.HostInfo{
		OS: runtime.GOOS, Architecture: runtime.GOARCH,
		LogicalCPUs: runtime.NumCPU(), Kernel: sysctlValue("kern.osrelease"),
		CPU: sysctlValue("machdep.cpu.brand_string"),
	}
	host.MemoryTotalBytes, host.MemoryBeforeBytes = Memory()
	host.ProcessesBefore = countProcesses()
	load := sysctlValue("vm.loadavg")
	host.LoadBefore = strings.Trim(load, "{} ")
	return host
}

func sysctlValue(name string) string {
	output, err := exec.Command("/usr/sbin/sysctl", "-n", name).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func Memory() (uint64, uint64) {
	total, _ := strconv.ParseUint(sysctlValue("hw.memsize"), 10, 64)
	output, err := exec.Command("/usr/bin/vm_stat").Output()
	if err != nil {
		return total, 0
	}
	pageSize, availablePages := uint64(4096), uint64(0)
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "page size of") {
			pageSize = parsePageSize(line, pageSize)
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok || !availablePageKind(strings.TrimSpace(key)) {
			continue
		}
		pages, _ := strconv.ParseUint(
			strings.Trim(strings.TrimSpace(value), "."), 10, 64,
		)
		availablePages += pages
	}
	return total, availablePages * pageSize
}

func parsePageSize(line string, fallback uint64) uint64 {
	fields := strings.Fields(line)
	for index, field := range fields {
		if field == "of" && index+1 < len(fields) {
			value, err := strconv.ParseUint(fields[index+1], 10, 64)
			if err == nil {
				return value
			}
		}
	}
	return fallback
}

func availablePageKind(name string) bool {
	switch name {
	case "Pages free", "Pages inactive", "Pages speculative", "Pages purgeable":
		return true
	default:
		return false
	}
}

func countProcesses() int {
	output, err := exec.Command("/bin/ps", "-axo", "pid=").Output()
	if err != nil {
		return 0
	}
	return len(strings.Fields(string(output)))
}

func ResourceUsage(state *os.ProcessState) (float64, float64, uint64) {
	var maxRSS uint64
	if usage, ok := state.SysUsage().(*syscall.Rusage); ok && usage.Maxrss > 0 {
		maxRSS = uint64(usage.Maxrss)
	}
	return state.UserTime().Seconds(), state.SystemTime().Seconds(), maxRSS
}
