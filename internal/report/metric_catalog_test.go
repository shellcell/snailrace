package report

import (
	"testing"

	"github.com/shellcell/snailrace/internal/model"
)

func TestMetricCatalogIdentityAndSelectors(t *testing.T) {
	if len(metricCatalog) != int(metricCount) {
		t.Fatalf("metric catalog length = %d, want %d", len(metricCatalog), metricCount)
	}
	run := model.Run{
		WallSeconds: 1, CPUUserSeconds: 2, CPUSystemSeconds: 3,
		AverageCPUPercent: 4, MeanResidentBytes: 5, PeakResidentBytes: 6,
		OSMaxRSSBytes: 7, PeakPhysicalFootprintBytes: 8,
		PhysicalFootprintValid: true, PeakVirtualBytes: 9,
		PeakProcesses: 10, PeakThreads: 11, PeakFileDescriptors: 12,
	}
	summary := model.Summarize([]model.Run{run})
	for index, metric := range metricCatalog {
		if metric.id != metricID(index) {
			t.Fatalf("metric %d has ID %d", index, metric.id)
		}
		if metric.name == "" || metric.chartName == "" || metric.format == nil {
			t.Fatalf("metric %d has incomplete presentation metadata", metric.id)
		}
		if got, want := metric.stats(summary).Mean, metric.run(run); got != want {
			t.Fatalf("metric %d summary = %v, run = %v", metric.id, got, want)
		}
	}
}

func TestChartGroupsReferenceEveryMetricOnce(t *testing.T) {
	groups := chartGroups(model.Report{Config: model.Config{Mode: "command"}})
	seen := make(map[metricID]int)
	for _, group := range groups {
		for _, metric := range group.metrics {
			seen[metric.id]++
		}
	}
	for _, metric := range metricCatalog {
		if seen[metric.id] != 1 {
			t.Fatalf("metric %d appears in %d chart groups", metric.id, seen[metric.id])
		}
	}
}
