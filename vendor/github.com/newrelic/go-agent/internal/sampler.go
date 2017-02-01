package internal

import (
	"runtime"
	"time"

	"github.com/newrelic/go-agent/internal/logger"
	"github.com/newrelic/go-agent/internal/sysinfo"
)

// Sample is a system/runtime snapshot.
type Sample struct {
	when         time.Time
	memStats     runtime.MemStats
	usage        sysinfo.Usage
	numGoroutine int
	numCPU       int
}

func bytesToMebibytesFloat(bts uint64) float64 {
	return float64(bts) / (1024 * 1024)
}

// GetSample gathers a new Sample.
func GetSample(now time.Time, lg logger.Logger) *Sample {
	s := Sample{
		when:         now,
		numGoroutine: runtime.NumGoroutine(),
		numCPU:       runtime.NumCPU(),
	}

	if usage, err := sysinfo.GetUsage(); err == nil {
		s.usage = usage
	} else {
		lg.Warn("unable to usage", map[string]interface{}{
			"error": err.Error(),
		})
	}

	runtime.ReadMemStats(&s.memStats)

	return &s
}

type cpuStats struct {
	used     time.Duration
	fraction float64 // used / (elapsed * numCPU)
}

// Stats contains system information for a period of time.
type Stats struct {
	numGoroutine    int
	allocBytes      uint64
	heapObjects     uint64
	user            cpuStats
	system          cpuStats
	gcPauseFraction float64
	deltaNumGC      uint32
	deltaPauseTotal time.Duration
	minPause        time.Duration
	maxPause        time.Duration
}

// Samples is used as the parameter to GetStats to avoid mixing up the previous
// and current sample.
type Samples struct {
	Previous *Sample
	Current  *Sample
}

// GetStats combines two Samples into a Stats.
func GetStats(ss Samples) Stats {
	cur := ss.Current
	prev := ss.Previous
	elapsed := cur.when.Sub(prev.when)

	s := Stats{
		numGoroutine: cur.numGoroutine,
		allocBytes:   cur.memStats.Alloc,
		heapObjects:  cur.memStats.HeapObjects,
	}

	// CPU Utilization
	totalCPUSeconds := elapsed.Seconds() * float64(cur.numCPU)
	if prev.usage.User != 0 && cur.usage.User > prev.usage.User {
		s.user.used = cur.usage.User - prev.usage.User
		s.user.fraction = s.user.used.Seconds() / totalCPUSeconds
	}
	if prev.usage.System != 0 && cur.usage.System > prev.usage.System {
		s.system.used = cur.usage.System - prev.usage.System
		s.system.fraction = s.system.used.Seconds() / totalCPUSeconds
	}

	// GC Pause Fraction
	deltaPauseTotalNs := cur.memStats.PauseTotalNs - prev.memStats.PauseTotalNs
	frac := float64(deltaPauseTotalNs) / float64(elapsed.Nanoseconds())
	s.gcPauseFraction = frac

	// GC Pauses
	if deltaNumGC := cur.memStats.NumGC - prev.memStats.NumGC; deltaNumGC > 0 {
		// In case more than 256 pauses have happened between samples
		// and we are examining a subset of the pauses, we ensure that
		// the min and max are not on the same side of the average by
		// using the average as the starting min and max.
		maxPauseNs := deltaPauseTotalNs / uint64(deltaNumGC)
		minPauseNs := deltaPauseTotalNs / uint64(deltaNumGC)
		for i := prev.memStats.NumGC + 1; i <= cur.memStats.NumGC; i++ {
			pause := cur.memStats.PauseNs[(i+255)%256]
			if pause > maxPauseNs {
				maxPauseNs = pause
			}
			if pause < minPauseNs {
				minPauseNs = pause
			}
		}
		s.deltaPauseTotal = time.Duration(deltaPauseTotalNs) * time.Nanosecond
		s.deltaNumGC = deltaNumGC
		s.minPause = time.Duration(minPauseNs) * time.Nanosecond
		s.maxPause = time.Duration(maxPauseNs) * time.Nanosecond
	}

	return s
}

// MergeIntoHarvest implements Harvestable.
func (s Stats) MergeIntoHarvest(h *Harvest) {
	h.Metrics.addValue(heapObjectsAllocated, "", float64(s.heapObjects), forced)
	h.Metrics.addValue(runGoroutine, "", float64(s.numGoroutine), forced)
	h.Metrics.addValueExclusive(memoryPhysical, "", bytesToMebibytesFloat(s.allocBytes), 0, forced)
	h.Metrics.addValueExclusive(cpuUserUtilization, "", s.user.fraction, 0, forced)
	h.Metrics.addValueExclusive(cpuSystemUtilization, "", s.system.fraction, 0, forced)
	h.Metrics.addValue(cpuUserTime, "", s.user.used.Seconds(), forced)
	h.Metrics.addValue(cpuSystemTime, "", s.system.used.Seconds(), forced)
	h.Metrics.addValueExclusive(gcPauseFraction, "", s.gcPauseFraction, 0, forced)
	if s.deltaNumGC > 0 {
		h.Metrics.add(gcPauses, "", metricData{
			countSatisfied:  float64(s.deltaNumGC),
			totalTolerated:  s.deltaPauseTotal.Seconds(),
			exclusiveFailed: 0,
			min:             s.minPause.Seconds(),
			max:             s.maxPause.Seconds(),
			sumSquares:      s.deltaPauseTotal.Seconds() * s.deltaPauseTotal.Seconds(),
		}, forced)
	}
}
