// +build linux freebsd darwin

package cpu

import (
	"fmt"
	"time"
)

func getAllBusy(t TimesStat) (float64, float64) {
	busy := t.User + t.System + t.Nice + t.Iowait + t.Irq +
		t.Softirq + t.Steal + t.Guest + t.GuestNice + t.Stolen
	return busy + t.Idle, busy
}

func calculateBusy(t1, t2 TimesStat) float64 {
	t1All, t1Busy := getAllBusy(t1)
	t2All, t2Busy := getAllBusy(t2)

	if t2Busy <= t1Busy {
		return 0
	}
	if t2All <= t1All {
		return 1
	}
	return (t2Busy - t1Busy) / (t2All - t1All) * 100
}

func calculateAllBusy(t1, t2 []TimesStat) ([]float64, error) {
	// Make sure the CPU measurements have the same length.
	if len(t1) != len(t2) {
		return nil, fmt.Errorf(
			"received two CPU counts: %d != %d",
			len(t1), len(t2),
		)
	}

	ret := make([]float64, len(t1))
	for i, t := range t2 {
		ret[i] = calculateBusy(t1[i], t)
	}
	return ret, nil
}

//Percent calculates the percentage of cpu used either per CPU or combined.
//If an interval of 0 is given it will compare the current cpu times against the last call.
func Percent(interval time.Duration, percpu bool) ([]float64, error) {
	if interval <= 0 {
		return percentUsedFromLastCall(percpu)
	}

	// Get CPU usage at the start of the interval.
	cpuTimes1, err := Times(percpu)
	if err != nil {
		return nil, err
	}

	time.Sleep(interval)

	// And at the end of the interval.
	cpuTimes2, err := Times(percpu)
	if err != nil {
		return nil, err
	}

	return calculateAllBusy(cpuTimes1, cpuTimes2)
}

func percentUsedFromLastCall(percpu bool) ([]float64, error) {
	cpuTimes, err := Times(percpu)
	if err != nil {
		return nil, err
	}
	lastCPUPercent.Lock()
	defer lastCPUPercent.Unlock()
	var lastTimes []TimesStat
	if percpu {
		lastTimes = lastCPUPercent.lastPerCPUTimes
		lastCPUPercent.lastPerCPUTimes = cpuTimes
	} else {
		lastTimes = lastCPUPercent.lastCPUTimes
		lastCPUPercent.lastCPUTimes = cpuTimes
	}

	if lastTimes == nil {
		return nil, fmt.Errorf("Error getting times for cpu percent. LastTimes was nil")
	}
	return calculateAllBusy(lastTimes, cpuTimes)

}
