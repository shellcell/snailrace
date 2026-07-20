package runner

import (
	"time"

	"github.com/shellcell/snailrace/internal/platform"
)

type processGroupSampler func(int) (platform.Metrics, bool)

type sampleAggregate struct {
	peak                  platform.Metrics
	footprintMissing      bool
	footprintInvalidated  bool
	residentByteSamples   uint64
	sampleCount           uint64
	residentByteSeconds   float64
	sampleCoverageSeconds float64
	previousRSS           uint64
	previousAt            time.Time
}

func (aggregate *sampleAggregate) observe(current platform.Metrics, at time.Time) {
	aggregate.addCoverage(at)
	aggregate.residentByteSamples += current.ResidentBytes
	aggregate.sampleCount++
	aggregate.peak.ResidentBytes = max(aggregate.peak.ResidentBytes, current.ResidentBytes)
	aggregate.peak.VirtualBytes = max(aggregate.peak.VirtualBytes, current.VirtualBytes)
	aggregate.peak.Processes = max(aggregate.peak.Processes, current.Processes)
	aggregate.peak.Threads = max(aggregate.peak.Threads, current.Threads)
	aggregate.peak.FileDescriptors = max(
		aggregate.peak.FileDescriptors, current.FileDescriptors,
	)
	aggregate.observeFootprint(current)
	aggregate.previousRSS, aggregate.previousAt = current.ResidentBytes, at
}

// The aggregate footprint is valid only when every sample carried a valid
// footprint and nothing invalidated it; a sample that merely lacks a footprint
// leaves the aggregate unusable without marking the run invalid.
func (aggregate *sampleAggregate) observeFootprint(current platform.Metrics) {
	if current.PhysicalFootprintValid {
		aggregate.peak.PhysicalFootprintBytes = max(
			aggregate.peak.PhysicalFootprintBytes, current.PhysicalFootprintBytes,
		)
	} else {
		aggregate.footprintMissing = true
	}
	if current.PhysicalFootprintInvalid {
		aggregate.footprintInvalidated = true
	}
	aggregate.peak.PhysicalFootprintValid = !aggregate.footprintMissing &&
		!aggregate.footprintInvalidated
	aggregate.peak.PhysicalFootprintInvalid = aggregate.footprintInvalidated
}

func (aggregate *sampleAggregate) invalidatePhysicalFootprint() {
	aggregate.footprintInvalidated = true
	aggregate.peak.PhysicalFootprintInvalid = true
	aggregate.peak.PhysicalFootprintValid = false
}

func (aggregate *sampleAggregate) addCoverage(at time.Time) {
	if aggregate.previousAt.IsZero() {
		return
	}
	seconds := at.Sub(aggregate.previousAt).Seconds()
	aggregate.residentByteSeconds += float64(aggregate.previousRSS) * seconds
	aggregate.sampleCoverageSeconds += seconds
}

func (aggregate sampleAggregate) meanResident() float64 {
	if aggregate.sampleCoverageSeconds > 0 {
		return aggregate.residentByteSeconds / aggregate.sampleCoverageSeconds
	}
	if aggregate.sampleCount == 0 {
		return 0
	}
	return float64(aggregate.residentByteSamples) / float64(aggregate.sampleCount)
}

type monitorHandle struct {
	stop chan struct{}
	done <-chan sampleAggregate
}

func startMonitor(groupID int, interval time.Duration, sample processGroupSampler) monitorHandle {
	stop := make(chan struct{})
	done := make(chan sampleAggregate, 1)
	go monitorProcessGroup(groupID, interval, sample, stop, done)
	return monitorHandle{stop: stop, done: done}
}

func (monitor monitorHandle) finish() sampleAggregate {
	close(monitor.stop)
	return <-monitor.done
}

func monitorProcessGroup(
	groupID int,
	interval time.Duration,
	sample processGroupSampler,
	stop <-chan struct{},
	done chan<- sampleAggregate,
) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	var aggregate sampleAggregate
	observe := func() {
		current, valid := sample(groupID)
		if !valid {
			if current.PhysicalFootprintInvalid {
				aggregate.invalidatePhysicalFootprint()
			}
			return
		}
		aggregate.observe(current, time.Now())
	}
	observe()
	for {
		select {
		case <-ticker.C:
			observe()
		case <-stop:
			aggregate.addCoverage(time.Now())
			done <- aggregate
			return
		}
	}
}
