package runner

import (
	"time"

	"github.com/shellcell/snailrace/internal/model"
)

type progressTracker struct {
	callback  func(ProgressEvent)
	started   time.Time
	completed int
	total     int
	tools     int
	fixed     time.Duration
	interval  float64
	runs      int
	estimates []ProgressEstimate
	summaries []progressSummary
	processed []int
}

func newProgressTracker(options Options, specs []Spec, config Config) *progressTracker {
	tracker := &progressTracker{
		callback: options.Progress, started: time.Now(),
		total: len(specs) * (config.Runs + config.Warmups),
		tools: len(specs), fixed: options.Duration,
		runs:     config.Runs,
		interval: float64(config.Interval) / float64(time.Millisecond),
	}
	if tracker.callback == nil {
		return tracker
	}
	tracker.estimates = make([]ProgressEstimate, len(specs))
	tracker.summaries = make([]progressSummary, len(specs))
	tracker.processed = make([]int, len(specs))
	for index, spec := range specs {
		tracker.estimates[index] = ProgressEstimate{ToolName: spec.Name, Total: config.Runs}
		tracker.summaries[index] = newProgressSummary(config.Runs)
	}
	return tracker
}

func (tracker *progressTracker) setTool(index int, tool model.ToolInfo) {
	if tracker.callback == nil {
		return
	}
	tracker.estimates[index].DiskFootprintBytes = tool.DiskFootprintBytes
}

func (tracker *progressTracker) update(
	tool int,
	name string,
	iteration, iterations int,
	warmup, completed bool,
	runs []model.Run,
) {
	if tracker.callback == nil {
		return
	}
	if completed {
		tracker.completed++
	}
	if !warmup {
		estimate := &tracker.estimates[tool]
		if len(runs) > 0 && (!estimate.HasEstimate || estimate.Completed != len(runs)) {
			if len(runs) < tracker.processed[tool] {
				tracker.summaries[tool] = newProgressSummary(tracker.runs)
				tracker.processed[tool] = 0
			}
			for _, run := range runs[tracker.processed[tool]:] {
				tracker.summaries[tool].add(run)
			}
			tracker.processed[tool] = len(runs)
			estimate.HasEstimate = true
			estimate.Estimate = tracker.summaries[tool].snapshot()
		}
		estimate.Completed = len(runs)
		estimate.Runs = runs
	}
	elapsed := time.Since(tracker.started)
	eta := time.Duration(0)
	if tracker.completed > 0 {
		perStep := elapsed / time.Duration(tracker.completed)
		eta = perStep * time.Duration(tracker.total-tracker.completed)
	}
	event := ProgressEvent{
		ToolName: name, Tool: tool + 1, ToolCount: tracker.tools,
		Iteration: iteration, Iterations: iterations,
		Completed: tracker.completed, Total: tracker.total, Warmup: warmup,
		FixedDuration: tracker.fixed, IntervalMS: tracker.interval,
		Elapsed: elapsed, ETA: eta,
		Estimates: append([]ProgressEstimate(nil), tracker.estimates...),
	}
	if len(runs) > 0 {
		event.HasEstimate = true
		event.Estimate = tracker.estimates[tool].Estimate
	}
	tracker.callback(event)
}

const (
	progressWall = iota
	progressCPUTotal
	progressCPUUser
	progressCPUSystem
	progressAverageCPU
	progressPeakResident
	progressOSMaxRSS
	progressMeanResident
	progressPeakVirtual
	progressPeakProcesses
	progressPeakThreads
	progressPeakFDs
	progressSamples
	progressCoverage
	progressMetricCount
)

type progressSummary struct {
	metrics  [progressMetricCount]model.RunningStats
	physical model.RunningStats
}

func newProgressSummary(capacity int) progressSummary {
	var summary progressSummary
	for index := range summary.metrics {
		summary.metrics[index] = model.NewRunningStats(capacity)
	}
	summary.physical = model.NewRunningStats(capacity)
	return summary
}

func (summary *progressSummary) add(run model.Run) {
	values := [...]float64{
		run.WallSeconds,
		run.CPUUserSeconds + run.CPUSystemSeconds,
		run.CPUUserSeconds,
		run.CPUSystemSeconds,
		run.AverageCPUPercent,
		run.PeakResidentBytes,
		run.OSMaxRSSBytes,
		run.MeanResidentBytes,
		run.PeakVirtualBytes,
		run.PeakProcesses,
		run.PeakThreads,
		run.PeakFileDescriptors,
		float64(run.SampleCount),
		run.SampleCoverageSeconds,
	}
	for index, value := range values {
		summary.metrics[index].Add(value)
	}
	if run.PhysicalFootprintValid {
		summary.physical.Add(run.PeakPhysicalFootprintBytes)
	}
}

func (summary progressSummary) snapshot() model.Summary {
	result := model.Summary{
		WallSeconds:           summary.metrics[progressWall].Snapshot(),
		CPUTotalSeconds:       summary.metrics[progressCPUTotal].Snapshot(),
		CPUUserSeconds:        summary.metrics[progressCPUUser].Snapshot(),
		CPUSystemSeconds:      summary.metrics[progressCPUSystem].Snapshot(),
		AverageCPUPercent:     summary.metrics[progressAverageCPU].Snapshot(),
		PeakResidentBytes:     summary.metrics[progressPeakResident].Snapshot(),
		OSMaxRSSBytes:         summary.metrics[progressOSMaxRSS].Snapshot(),
		MeanResidentBytes:     summary.metrics[progressMeanResident].Snapshot(),
		PeakVirtualBytes:      summary.metrics[progressPeakVirtual].Snapshot(),
		PeakProcesses:         summary.metrics[progressPeakProcesses].Snapshot(),
		PeakThreads:           summary.metrics[progressPeakThreads].Snapshot(),
		PeakFileDescriptors:   summary.metrics[progressPeakFDs].Snapshot(),
		ValidSampleCount:      summary.metrics[progressSamples].Snapshot(),
		SampleCoverageSeconds: summary.metrics[progressCoverage].Snapshot(),
	}
	physical := summary.physical.Snapshot()
	if physical.N > 0 {
		result.PeakPhysicalFootprintBytes = &physical
	}
	return result
}

func (tracker *progressTracker) finish() {
	if tracker.callback != nil {
		tracker.callback(ProgressEvent{Finished: true})
	}
}
