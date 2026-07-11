package platform

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func DefaultInterval() time.Duration { return 10 * time.Millisecond }

func SampleImmediately() bool { return true }

func SampleTree(rootPID int) (Metrics, bool) {
	seen := make(map[int]bool)
	queue := []int{rootPID}
	var total Metrics
	for len(queue) > 0 {
		pid := queue[0]
		queue = queue[1:]
		if seen[pid] {
			continue
		}
		seen[pid] = true
		record, err := readProcess(pid)
		if err != nil {
			if pid == rootPID {
				return Metrics{}, false
			}
			continue
		}
		total.ResidentBytes += record.ResidentBytes
		total.VirtualBytes += record.VirtualBytes
		total.Processes++
		total.Threads += record.Threads
		total.FileDescriptors += record.FileDescriptors
		queue = append(queue, readChildren(pid)...)
	}
	return total, true
}

func readChildren(pid int) []int {
	taskDirectory := filepath.Join("/proc", strconv.Itoa(pid), "task")
	tasks, err := os.ReadDir(taskDirectory)
	if err != nil {
		return nil
	}
	seen := make(map[int]bool)
	children := make([]int, 0, 2)
	for _, task := range tasks {
		data, readErr := os.ReadFile(filepath.Join(taskDirectory, task.Name(), "children"))
		if readErr != nil {
			continue
		}
		for _, field := range strings.Fields(string(data)) {
			child, parseErr := strconv.Atoi(field)
			if parseErr == nil && !seen[child] {
				seen[child] = true
				children = append(children, child)
			}
		}
	}
	return children
}

func readProcess(pid int) (Process, error) {
	base := filepath.Join("/proc", strconv.Itoa(pid))
	data, err := os.ReadFile(filepath.Join(base, "stat"))
	if err != nil {
		return Process{}, err
	}
	closeParen := strings.LastIndexByte(string(data), ')')
	if closeParen < 0 || closeParen+2 >= len(data) {
		return Process{}, errors.New("malformed process stat")
	}
	fields := strings.Fields(string(data[closeParen+2:]))
	if len(fields) < 22 {
		return Process{}, errors.New("short process stat")
	}
	ppid, _ := strconv.Atoi(fields[1])
	threads, _ := strconv.ParseUint(fields[17], 10, 64)
	virtual, _ := strconv.ParseUint(fields[20], 10, 64)
	rssPages, _ := strconv.ParseInt(fields[21], 10, 64)
	if rssPages < 0 {
		rssPages = 0
	}
	return Process{
		PID: pid, PPID: ppid, Threads: threads, VirtualBytes: virtual,
		ResidentBytes:   uint64(rssPages) * uint64(os.Getpagesize()),
		FileDescriptors: countDirectory(filepath.Join(base, "fd")),
	}, nil
}

func countDirectory(path string) uint64 {
	directory, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer directory.Close()
	var count uint64
	for {
		entries, readErr := directory.ReadDir(128)
		count += uint64(len(entries))
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				return count
			}
			return count
		}
	}
}
