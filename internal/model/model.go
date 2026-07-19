package model

import "time"

type Config struct {
	Runs              int           `json:"runs"`
	Warmups           int           `json:"warmups"`
	Interval          time.Duration `json:"-"`
	IntervalMS        float64       `json:"sample_interval_ms"`
	Mode              string        `json:"mode"`
	DurationSeconds   float64       `json:"duration_seconds,omitempty"`
	TerminalWidth     uint16        `json:"terminal_width,omitempty"`
	TerminalHeight    uint16        `json:"terminal_height,omitempty"`
	TerminalInherited bool          `json:"terminal_inherited,omitempty"`
	Baseline          int           `json:"baseline"`
	BaselineAutomatic bool          `json:"baseline_automatic"`
	OrderSeed         int64         `json:"order_seed"`
	OrderMethod       string        `json:"order_method"`
	IndexDimensions   []string      `json:"index_dimensions"`
	MeasurementOrder  [][]int       `json:"measurement_order"`
	WarmupOrder       [][]int       `json:"warmup_order,omitempty"`
	OutputMode        string        `json:"output_mode"`
}

type HostInfo struct {
	OS                string `json:"os"`
	Architecture      string `json:"architecture"`
	Kernel            string `json:"kernel"`
	CPU               string `json:"cpu"`
	LogicalCPUs       int    `json:"logical_cpus"`
	MemoryTotalBytes  uint64 `json:"memory_total_bytes"`
	MemoryBeforeBytes uint64 `json:"memory_available_before_bytes"`
	MemoryAfterBytes  uint64 `json:"memory_available_after_bytes"`
	ProcessesBefore   int    `json:"processes_before"`
	LoadBefore        string `json:"load_average_before"`
}

type ToolInfo struct {
	Name               string     `json:"name"`
	Command            []string   `json:"command"`
	Executable         string     `json:"executable,omitempty"`
	SizeBytes          int64      `json:"size_bytes,omitempty"`
	SHA256             string     `json:"sha256,omitempty"`
	LinkedSizeBytes    int64      `json:"linked_size_bytes,omitempty"`
	DiskFootprintBytes int64      `json:"disk_footprint_bytes,omitempty"`
	LinkedFiles        []DiskFile `json:"linked_files,omitempty"`
	SharedCacheFiles   []string   `json:"dyld_shared_cache_files,omitempty"`
}

type DiskFile struct {
	Path      string `json:"path"`
	SizeBytes int64  `json:"size_bytes"`
}

type Run struct {
	Index                      int     `json:"index"`
	ExitCode                   int     `json:"exit_code"`
	WallSeconds                float64 `json:"wall_seconds"`
	CPUUserSeconds             float64 `json:"cpu_user_seconds"`
	CPUSystemSeconds           float64 `json:"cpu_system_seconds"`
	AverageCPUPercent          float64 `json:"average_cpu_percent"`
	PeakResidentBytes          float64 `json:"peak_resident_bytes"`
	PeakPhysicalFootprintBytes float64 `json:"peak_physical_footprint_bytes,omitempty"`
	OSMaxRSSBytes              float64 `json:"os_max_rss_bytes"`
	MeanResidentBytes          float64 `json:"mean_resident_bytes"`
	PeakVirtualBytes           float64 `json:"peak_virtual_bytes"`
	PeakProcesses              float64 `json:"peak_processes"`
	PeakThreads                float64 `json:"peak_threads"`
	PeakFileDescriptors        float64 `json:"peak_file_descriptors"`
	StopReason                 string  `json:"stop_reason"`
	SampleCount                int     `json:"valid_sample_count"`
	SampleCoverageSeconds      float64 `json:"sample_coverage_seconds"`
}

func (run Run) Failed() bool {
	return run.StopReason != "duration" && run.ExitCode != 0
}

type Stats struct {
	N         int     `json:"n"`
	Min       float64 `json:"min"`
	Max       float64 `json:"max"`
	Mean      float64 `json:"mean"`
	StdDev    float64 `json:"stddev"`
	Median    float64 `json:"median"`
	P95       float64 `json:"p95"`
	CI95Low   float64 `json:"-"`
	CI95High  float64 `json:"-"`
	CI95Valid bool    `json:"ci95_valid"`
}

type Summary struct {
	WallSeconds                Stats  `json:"wall_seconds"`
	CPUTotalSeconds            Stats  `json:"cpu_total_seconds"`
	CPUUserSeconds             Stats  `json:"cpu_user_seconds"`
	CPUSystemSeconds           Stats  `json:"cpu_system_seconds"`
	AverageCPUPercent          Stats  `json:"average_cpu_percent"`
	PeakResidentBytes          Stats  `json:"peak_resident_bytes"`
	PeakPhysicalFootprintBytes *Stats `json:"peak_physical_footprint_bytes,omitempty"`
	OSMaxRSSBytes              Stats  `json:"os_max_rss_bytes"`
	MeanResidentBytes          Stats  `json:"mean_resident_bytes"`
	PeakVirtualBytes           Stats  `json:"peak_virtual_bytes"`
	PeakProcesses              Stats  `json:"peak_processes"`
	PeakThreads                Stats  `json:"peak_threads"`
	PeakFileDescriptors        Stats  `json:"peak_file_descriptors"`
	ValidSampleCount           Stats  `json:"valid_sample_count"`
	SampleCoverageSeconds      Stats  `json:"sample_coverage_seconds"`
}

func (summary Summary) PhysicalFootprintStats() Stats {
	if summary.PeakPhysicalFootprintBytes == nil {
		return Stats{}
	}
	return *summary.PeakPhysicalFootprintBytes
}

type Benchmark struct {
	Tool    ToolInfo `json:"tool"`
	Runs    []Run    `json:"runs"`
	Summary Summary  `json:"summary"`
}

func (benchmark Benchmark) FailedRunCount() int {
	count := 0
	for _, run := range benchmark.Runs {
		if run.Failed() {
			count++
		}
	}
	return count
}

func (benchmark Benchmark) EligibleForRanking() bool {
	return len(benchmark.Runs) > 0 && benchmark.FailedRunCount() == 0
}

type Report struct {
	MeasuredAt time.Time   `json:"measured_at"`
	Config     Config      `json:"config"`
	Host       HostInfo    `json:"host"`
	Benchmarks []Benchmark `json:"benchmarks"`
	Notes      []string    `json:"notes,omitempty"`
	Verbose    bool        `json:"-"`
}
