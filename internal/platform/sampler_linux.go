package platform

import (
	"bytes"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"golang.org/x/sys/unix"
)

func DefaultInterval() time.Duration { return 10 * time.Millisecond }

func SampleImmediately() bool { return true }

func SampleTree(rootPID int) (Metrics, bool) {
	directory, err := unix.Open("/proc", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return Metrics{}, false
	}
	defer unix.Close(directory)
	var total Metrics
	rootFound := false
	statBuffer := make([]byte, 4096)
	directoryBuffer := make([]byte, 32*1024)
	names := make([]string, 0, 256)
	for {
		count, readErr := unix.ReadDirent(directory, directoryBuffer)
		if readErr != nil {
			return Metrics{}, false
		}
		if count == 0 {
			return total, rootFound
		}
		_, _, names = unix.ParseDirent(directoryBuffer[:count], -1, names[:0])
		for _, name := range names {
			pid, ok := parsePID(name)
			if !ok {
				continue
			}
			record, err := readProcessAt(
				directory, name, pid, rootPID, statBuffer,
			)
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
	}
}

func readProcess(pid int) (Process, error) {
	base := filepath.Join("/proc", strconv.Itoa(pid))
	data, err := os.ReadFile(filepath.Join(base, "stat"))
	if err != nil {
		return Process{}, err
	}
	process, err := parseProcessStat(data, pid)
	if err != nil {
		return Process{}, err
	}
	process.FileDescriptors = countDirectory(filepath.Join(base, "fd"))
	return process, nil
}

func readProcessAt(
	procFD int, name string, pid, groupID int, buffer []byte,
) (Process, error) {
	file, err := unix.Openat(procFD, name+"/stat", unix.O_RDONLY|unix.O_CLOEXEC, 0)
	if err != nil {
		return Process{}, err
	}
	count, readErr := unix.Read(file, buffer)
	_ = unix.Close(file)
	if readErr != nil {
		return Process{}, readErr
	}
	if count == len(buffer) {
		return Process{}, errors.New("process stat exceeds buffer")
	}
	process, err := parseProcessStat(buffer[:count], pid)
	if err != nil {
		return Process{}, err
	}
	if process.GroupID == groupID {
		process.FileDescriptors = countDirectory("/proc/" + name + "/fd")
	}
	return process, nil
}

func parseProcessStat(data []byte, pid int) (Process, error) {
	closeParen := bytes.LastIndexByte(data, ')')
	if closeParen < 0 || closeParen+2 >= len(data) {
		return Process{}, errors.New("malformed process stat")
	}
	fields := data[closeParen+2:]
	var process Process
	process.PID = pid
	field := 0
	for offset := 0; offset < len(fields); {
		for offset < len(fields) && (fields[offset] == ' ' || fields[offset] == '\n') {
			offset++
		}
		start := offset
		for offset < len(fields) && fields[offset] != ' ' && fields[offset] != '\n' {
			offset++
		}
		if start == offset {
			break
		}
		value := fields[start:offset]
		var err error
		switch field {
		case 1:
			process.PPID, err = parsePositiveInt(value)
		case 2:
			process.GroupID, err = parsePositiveInt(value)
		case 17:
			process.Threads, err = parseUint(value)
		case 20:
			process.VirtualBytes, err = parseUint(value)
		case 21:
			var pages int64
			pages, err = parseInt64(value)
			if pages > 0 {
				pageSize := uint64(os.Getpagesize())
				if uint64(pages) > math.MaxUint64/pageSize {
					err = errors.New("resident size overflows")
				} else {
					process.ResidentBytes = uint64(pages) * pageSize
				}
			}
		}
		if err != nil {
			return Process{}, err
		}
		field++
		if field > 21 {
			return process, nil
		}
	}
	return Process{}, errors.New("short process stat")
}

func parsePID(value string) (int, bool) {
	if value == "" {
		return 0, false
	}
	result := 0
	for index := range len(value) {
		digit := value[index] - '0'
		if digit > 9 || result > (math.MaxInt-int(digit))/10 {
			return 0, false
		}
		result = result*10 + int(digit)
	}
	return result, result > 0
}

func parsePositiveInt(value []byte) (int, error) {
	parsed, err := parseUint(value)
	if err != nil || parsed > uint64(math.MaxInt) {
		return 0, errors.New("invalid process integer")
	}
	return int(parsed), nil
}

func parseUint(value []byte) (uint64, error) {
	if len(value) == 0 {
		return 0, errors.New("empty process integer")
	}
	var result uint64
	for _, character := range value {
		digit := character - '0'
		if digit > 9 || result > (math.MaxUint64-uint64(digit))/10 {
			return 0, errors.New("invalid process integer")
		}
		result = result*10 + uint64(digit)
	}
	return result, nil
}

func parseInt64(value []byte) (int64, error) {
	if len(value) == 0 {
		return 0, errors.New("empty process integer")
	}
	negative := value[0] == '-'
	if negative {
		value = value[1:]
	}
	parsed, err := parseUint(value)
	if err != nil || (!negative && parsed > math.MaxInt64) ||
		(negative && parsed > uint64(math.MaxInt64)+1) {
		return 0, errors.New("invalid process integer")
	}
	if negative {
		if parsed == uint64(math.MaxInt64)+1 {
			return math.MinInt64, nil
		}
		return -int64(parsed), nil
	}
	return int64(parsed), nil
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
