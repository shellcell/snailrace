package platform

import (
	"bytes"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"unsafe"

	"golang.org/x/sys/unix"
)

func SampleProcessGroup(rootPID int) (Metrics, bool) {
	directory, err := unix.Open("/proc", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return Metrics{}, false
	}
	defer unix.Close(directory)
	var total Metrics
	rootFound := false
	statBuffer := make([]byte, 4096)
	directoryBuffer := make([]byte, 32*1024)
	for {
		count, readErr := unix.ReadDirent(directory, directoryBuffer)
		if readErr != nil {
			return Metrics{}, false
		}
		if count == 0 {
			return total, rootFound
		}
		validDirents := scanProcDirents(directoryBuffer[:count], func(pid int) {
			record, err := readProcessAt(
				directory, pid, rootPID, statBuffer,
			)
			if err != nil || record.GroupID != rootPID {
				return
			}
			if record.PID == rootPID {
				rootFound = true
			}
			total.ResidentBytes += record.ResidentBytes
			total.VirtualBytes += record.VirtualBytes
			total.Processes++
			total.Threads += record.Threads
			total.FileDescriptors += record.FileDescriptors
		})
		if !validDirents {
			return Metrics{}, false
		}
	}
}

func readProcess(pid int) (process, error) {
	base := filepath.Join("/proc", strconv.Itoa(pid))
	data, err := os.ReadFile(filepath.Join(base, "stat"))
	if err != nil {
		return process{}, err
	}
	record, err := parseProcessStat(data, pid)
	if err != nil {
		return process{}, err
	}
	record.FileDescriptors = countDirectory(filepath.Join(base, "fd"))
	return record, nil
}

func readProcessAt(
	procFD, pid, groupID int, buffer []byte,
) (process, error) {
	file, err := openProcessStat(procFD, pid)
	if err != nil {
		return process{}, err
	}
	count, readErr := unix.Read(file, buffer)
	_ = unix.Close(file)
	if readErr != nil {
		return process{}, readErr
	}
	if count == len(buffer) {
		return process{}, errors.New("process stat exceeds buffer")
	}
	record, err := parseProcessStat(buffer[:count], pid)
	if err != nil {
		return process{}, err
	}
	if record.GroupID == groupID {
		record.FileDescriptors = countDirectory(
			"/proc/" + strconv.Itoa(pid) + "/fd",
		)
	}
	return record, nil
}

func openProcessStat(procFD, pid int) (int, error) {
	var storage [32]byte
	path := strconv.AppendInt(storage[:0], int64(pid), 10)
	path = append(path, '/', 's', 't', 'a', 't', 0)
	file, _, errno := unix.Syscall6(
		unix.SYS_OPENAT,
		uintptr(procFD), uintptr(unsafe.Pointer(&path[0])),
		uintptr(unix.O_RDONLY|unix.O_CLOEXEC), 0, 0, 0,
	)
	runtime.KeepAlive(path)
	if errno != 0 {
		return -1, errno
	}
	return int(file), nil
}

func parseProcessStat(data []byte, pid int) (process, error) {
	closeParen := bytes.LastIndexByte(data, ')')
	if closeParen < 0 || closeParen+2 >= len(data) {
		return process{}, errors.New("malformed process stat")
	}
	fields := data[closeParen+2:]
	var result process
	result.PID = pid
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
			result.PPID, err = parsePositiveInt(value)
		case 2:
			result.GroupID, err = parsePositiveInt(value)
		case 17:
			result.Threads, err = parseUint(value)
		case 20:
			result.VirtualBytes, err = parseUint(value)
		case 21:
			var pages int64
			pages, err = parseInt64(value)
			if pages > 0 {
				pageSize := uint64(os.Getpagesize())
				if uint64(pages) > math.MaxUint64/pageSize {
					err = errors.New("resident size overflows")
				} else {
					result.ResidentBytes = uint64(pages) * pageSize
				}
			}
		}
		if err != nil {
			return process{}, err
		}
		field++
		if field > 21 {
			return result, nil
		}
	}
	return process{}, errors.New("short process stat")
}

func scanProcDirents(data []byte, visit func(int)) bool {
	// linux_dirent64 stores d_reclen at byte 16 and d_name at byte 19.
	const recordHeader = 19
	for offset := 0; offset < len(data); {
		if len(data)-offset < recordHeader {
			return false
		}
		recordLength := int(data[offset+16]) | int(data[offset+17])<<8
		if recordLength < recordHeader || recordLength%8 != 0 ||
			recordLength > len(data)-offset {
			return false
		}
		name := data[offset+recordHeader : offset+recordLength]
		nameLength := bytes.IndexByte(name, 0)
		if nameLength < 0 {
			return false
		}
		if pid, ok := parsePIDBytes(name[:nameLength]); ok {
			visit(pid)
		}
		offset += recordLength
	}
	return true
}

func parsePIDBytes(value []byte) (int, bool) {
	if len(value) == 0 {
		return 0, false
	}
	result := 0
	for _, character := range value {
		digit := character - '0'
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
