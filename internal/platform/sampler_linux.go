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
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return Metrics{}, false
	}
	var total Metrics
	rootFound := false
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		record, err := readProcess(pid)
		if err != nil || record.GroupID != rootPID {
			continue
		}
		if record.PID == rootPID {
			rootFound = true
		}
		total.ResidentBytes += record.ResidentBytes
		total.VirtualBytes += record.VirtualBytes
		total.Processes++
		total.Threads += record.Threads
		total.FileDescriptors += record.FileDescriptors
	}
	return total, rootFound
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
	groupID, _ := strconv.Atoi(fields[2])
	threads, _ := strconv.ParseUint(fields[17], 10, 64)
	virtual, _ := strconv.ParseUint(fields[20], 10, 64)
	rssPages, _ := strconv.ParseInt(fields[21], 10, 64)
	if rssPages < 0 {
		rssPages = 0
	}
	return Process{
		PID: pid, PPID: ppid, GroupID: groupID, Threads: threads, VirtualBytes: virtual,
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
