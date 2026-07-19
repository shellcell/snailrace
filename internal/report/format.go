package report

import (
	"fmt"
	"math"
	"strings"

	"github.com/shellcell/snailrace/internal/model"
)

type metricRow struct {
	name              string
	stats             func(model.Summary) model.Stats
	format            func(float64) string
	unavailableDarwin bool
	darwinOnly        bool
	run               func(model.Run) float64
	direction         metricDirection
	sampled           bool
}

var metricRows = []metricRow{
	{"Wall time", func(s model.Summary) model.Stats { return s.WallSeconds }, formatDuration, false, false,
		func(r model.Run) float64 { return r.WallSeconds }, lowerIsBetter, false},
	{"CPU total", func(s model.Summary) model.Stats { return s.CPUTotalSeconds }, formatDuration, false, false,
		func(r model.Run) float64 { return r.CPUUserSeconds + r.CPUSystemSeconds }, lowerIsBetter, false},
	{"CPU user", func(s model.Summary) model.Stats { return s.CPUUserSeconds }, formatDuration, false, false,
		func(r model.Run) float64 { return r.CPUUserSeconds }, lowerIsBetter, false},
	{"CPU system", func(s model.Summary) model.Stats { return s.CPUSystemSeconds }, formatDuration, false, false,
		func(r model.Run) float64 { return r.CPUSystemSeconds }, lowerIsBetter, false},
	{"Average CPU", func(s model.Summary) model.Stats { return s.AverageCPUPercent }, formatPercent, false, false,
		func(r model.Run) float64 { return r.AverageCPUPercent }, neutralDirection, false},
	{"Mean resident", func(s model.Summary) model.Stats { return s.MeanResidentBytes }, formatBytes, false, false,
		func(r model.Run) float64 { return r.MeanResidentBytes }, lowerIsBetter, true},
	{"Peak resident", func(s model.Summary) model.Stats { return s.PeakResidentBytes }, formatBytes, false, false,
		func(r model.Run) float64 { return r.PeakResidentBytes }, lowerIsBetter, true},
	{"OS-reported max RSS", func(s model.Summary) model.Stats { return s.OSMaxRSSBytes }, formatBytes, false, false,
		func(r model.Run) float64 { return r.OSMaxRSSBytes }, lowerIsBetter, false},
	{"Peak physical footprint", func(s model.Summary) model.Stats { return s.PhysicalFootprintStats() }, formatBytes, false, true,
		func(r model.Run) float64 { return r.PeakPhysicalFootprintBytes }, lowerIsBetter, true},
	{"Peak virtual", func(s model.Summary) model.Stats { return s.PeakVirtualBytes }, formatBytes, false, false,
		func(r model.Run) float64 { return r.PeakVirtualBytes }, lowerIsBetter, true},
	{"Peak processes", func(s model.Summary) model.Stats { return s.PeakProcesses }, formatCount, false, false,
		func(r model.Run) float64 { return r.PeakProcesses }, neutralDirection, true},
	{"Peak threads", func(s model.Summary) model.Stats { return s.PeakThreads }, formatCount, false, false,
		func(r model.Run) float64 { return r.PeakThreads }, neutralDirection, true},
	{"Peak FD references", func(s model.Summary) model.Stats { return s.PeakFileDescriptors }, formatCount, true, false,
		func(r model.Run) float64 { return r.PeakFileDescriptors }, lowerIsBetter, true},
}

func formatDuration(seconds float64) string {
	switch {
	case math.Abs(seconds) < 1e-6:
		return formatNumber(seconds*1e9) + " ns"
	case math.Abs(seconds) < 1e-3:
		return formatNumber(seconds*1e6) + " us"
	case math.Abs(seconds) < 1:
		return formatNumber(seconds*1e3) + " ms"
	default:
		return formatNumber(seconds) + " s"
	}
}

func formatBytes(value float64) string {
	units := [...]string{"B", "KiB", "MiB", "GiB", "TiB"}
	index := 0
	for math.Abs(value) >= 1024 && index < len(units)-1 {
		value /= 1024
		index++
	}
	return formatNumber(value) + " " + units[index]
}

func formatCount(value float64) string { return formatNumber(value) }

func formatPercent(value float64) string { return formatNumber(value) + "%" }

func formatConfidence(stats model.Stats, format func(float64) string) string {
	if !stats.CI95Valid {
		return "N/A (n < 2)"
	}
	return "[" + format(stats.CI95Low) + ", " + format(stats.CI95High) + "]"
}

func formatSignedPercent(value float64) string {
	prefix := "+"
	if value < 0 {
		prefix = "-"
		value = -value
	}
	return prefix + formatNumber(value) + "%"
}

func formatNumber(value float64) string {
	if value == 0 {
		return "0"
	}
	absolute := math.Abs(value)
	format := "%.3f"
	switch {
	case absolute >= 100:
		format = "%.0f"
	case absolute >= 10:
		format = "%.1f"
	case absolute >= 1:
		format = "%.2f"
	case absolute < 0.1:
		return fmt.Sprintf("%.3g", value)
	}
	result := fmt.Sprintf(format, value)
	if strings.Contains(result, ".") {
		result = strings.TrimRight(result, "0")
		result = strings.TrimRight(result, ".")
	}
	return result
}

func available(row metricRow, operatingSystem string) bool {
	return (operatingSystem != "darwin" || !row.unavailableDarwin) &&
		(!row.darwinOnly || operatingSystem == "darwin")
}

func availableFor(
	row metricRow, operatingSystem string, summaries ...model.Summary,
) bool {
	if !available(row, operatingSystem) {
		return false
	}
	if !row.darwinOnly {
		return true
	}
	for _, summary := range summaries {
		if row.stats(summary).N == 0 {
			return false
		}
	}
	return true
}
