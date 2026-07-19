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
	estimates []ProgressEstimate
}

func newProgressTracker(options Options, specs []Spec, config Config) *progressTracker {
	tracker := &progressTracker{
		callback: options.Progress, started: time.Now(),
		total: len(specs) * (config.Runs + config.Warmups),
		tools: len(specs), fixed: options.Duration,
		interval: float64(config.Interval) / float64(time.Millisecond),
	}
	if tracker.callback == nil {
		return tracker
	}
	tracker.estimates = make([]ProgressEstimate, len(specs))
	for index, spec := range specs {
		tracker.estimates[index] = ProgressEstimate{ToolName: spec.Name, Total: config.Runs}
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
			estimate.HasEstimate = true
			estimate.Estimate = model.Summarize(runs)
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

func (tracker *progressTracker) finish() {
	if tracker.callback != nil {
		tracker.callback(ProgressEvent{Finished: true})
	}
}
