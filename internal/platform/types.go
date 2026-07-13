package platform

type Metrics struct {
	ResidentBytes          uint64
	PhysicalFootprintBytes uint64
	VirtualBytes           uint64
	Processes              uint64
	Threads                uint64
	FileDescriptors        uint64
	ResidentByteSamples    uint64
	SampleCount            uint64
	ResidentByteSeconds    float64
	SampleCoverageSeconds  float64
}

type Process struct {
	PID, PPID                   int
	ResidentBytes, VirtualBytes uint64
	Threads, FileDescriptors    uint64
}

type LinkedDependency struct {
	Path        string
	SharedCache bool
}
