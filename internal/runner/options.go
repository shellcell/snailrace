package runner

import (
	"io"
	"time"

	"github.com/shellcell/snailrace/internal/model"
)

type Config struct {
	Runs, Warmups    int
	Interval         time.Duration
	OrderSeed        int64
	MeasurementOrder [][]int
	WarmupOrder      [][]int
}

type Options struct {
	ShowOutput   bool
	Output       io.Writer
	Input        io.Reader
	TerminalOut  io.Writer
	TUI          bool
	Duration     time.Duration
	Width        uint16
	Height       uint16
	Interactive  bool
	FollowResize bool
	Progress     func(ProgressEvent)
}

type ProgressEvent struct {
	ToolName      string
	Tool          int
	ToolCount     int
	Iteration     int
	Iterations    int
	Completed     int
	Total         int
	Warmup        bool
	Finished      bool
	HasEstimate   bool
	Estimate      model.Summary
	Estimates     []ProgressEstimate
	FixedDuration time.Duration
	IntervalMS    float64
	Elapsed       time.Duration
	ETA           time.Duration
}

type ProgressEstimate struct {
	ToolName           string
	Completed          int
	Total              int
	HasEstimate        bool
	Estimate           model.Summary
	Runs               []model.Run
	DiskFootprintBytes int64
}
