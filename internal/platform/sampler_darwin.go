package platform

import (
	"bufio"
	"bytes"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func DefaultInterval() time.Duration { return 50 * time.Millisecond }

func SampleImmediately() bool { return false }

func SampleTree(rootPID int) (Metrics, bool) {
	output, err := exec.Command(
		"/bin/ps", "-axo", "pid=,ppid=,rss=,vsz=,thcount=",
	).Output()
	if err != nil {
		return Metrics{}, false
	}
	records := parseProcesses(output)
	children := make(map[int][]int, len(records))
	for pid, record := range records {
		children[record.PPID] = append(children[record.PPID], pid)
	}
	var total Metrics
	foundRoot := false
	seen := make(map[int]bool)
	queue := []int{rootPID}
	for len(queue) > 0 {
		pid := queue[0]
		queue = queue[1:]
		if seen[pid] {
			continue
		}
		seen[pid] = true
		record, ok := records[pid]
		if !ok {
			continue
		}
		if pid == rootPID {
			foundRoot = true
		}
		total.ResidentBytes += record.ResidentBytes
		total.VirtualBytes += record.VirtualBytes
		total.Processes++
		total.Threads += record.Threads
		queue = append(queue, children[pid]...)
	}
	return total, foundRoot
}

func parseProcesses(output []byte) map[int]Process {
	records := make(map[int]Process)
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) != 5 {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		ppid, _ := strconv.Atoi(fields[1])
		rss, _ := strconv.ParseUint(fields[2], 10, 64)
		vsz, _ := strconv.ParseUint(fields[3], 10, 64)
		threads, _ := strconv.ParseUint(fields[4], 10, 64)
		records[pid] = Process{
			PID: pid, PPID: ppid, ResidentBytes: rss * 1024,
			VirtualBytes: vsz * 1024, Threads: threads,
		}
	}
	return records
}
