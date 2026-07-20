package platform

import "time"

func DefaultInterval() time.Duration { return 10 * time.Millisecond }

type Metrics struct {
	ResidentBytes            uint64
	PhysicalFootprintBytes   uint64
	PhysicalFootprintValid   bool
	PhysicalFootprintInvalid bool
	VirtualBytes             uint64
	Processes                uint64
	Threads                  uint64
	FileDescriptors          uint64
}

type process struct {
	PID, PPID, GroupID          int
	ResidentBytes, VirtualBytes uint64
	Threads, FileDescriptors    uint64
}

type LinkedDependency struct {
	Path        string
	SharedCache bool
}
