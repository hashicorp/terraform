// +build darwin

package cpu

import (
	"os/exec"
	"strconv"
	"strings"
)

// sys/resource.h
const (
	CPUser    = 0
	CPNice    = 1
	CPSys     = 2
	CPIntr    = 3
	CPIdle    = 4
	CPUStates = 5
)

// default value. from time.h
var ClocksPerSec = float64(128)

func Times(percpu bool) ([]TimesStat, error) {
	if percpu {
		return perCPUTimes()
	}

	return allCPUTimes()
}

// Returns only one CPUInfoStat on FreeBSD
func Info() ([]InfoStat, error) {
	var ret []InfoStat
	sysctl, err := exec.LookPath("/usr/sbin/sysctl")
	if err != nil {
		return ret, err
	}
	out, err := invoke.Command(sysctl, "machdep.cpu")
	if err != nil {
		return ret, err
	}

	c := InfoStat{}
	for _, line := range strings.Split(string(out), "\n") {
		values := strings.Fields(line)
		if len(values) < 1 {
			continue
		}

		t, err := strconv.ParseInt(values[1], 10, 64)
		// err is not checked here because some value is string.
		if strings.HasPrefix(line, "machdep.cpu.brand_string") {
			c.ModelName = strings.Join(values[1:], " ")
		} else if strings.HasPrefix(line, "machdep.cpu.family") {
			c.Family = values[1]
		} else if strings.HasPrefix(line, "machdep.cpu.model") {
			c.Model = values[1]
		} else if strings.HasPrefix(line, "machdep.cpu.stepping") {
			if err != nil {
				return ret, err
			}
			c.Stepping = int32(t)
		} else if strings.HasPrefix(line, "machdep.cpu.features") {
			for _, v := range values[1:] {
				c.Flags = append(c.Flags, strings.ToLower(v))
			}
		} else if strings.HasPrefix(line, "machdep.cpu.leaf7_features") {
			for _, v := range values[1:] {
				c.Flags = append(c.Flags, strings.ToLower(v))
			}
		} else if strings.HasPrefix(line, "machdep.cpu.extfeatures") {
			for _, v := range values[1:] {
				c.Flags = append(c.Flags, strings.ToLower(v))
			}
		} else if strings.HasPrefix(line, "machdep.cpu.core_count") {
			if err != nil {
				return ret, err
			}
			c.Cores = int32(t)
		} else if strings.HasPrefix(line, "machdep.cpu.cache.size") {
			if err != nil {
				return ret, err
			}
			c.CacheSize = int32(t)
		} else if strings.HasPrefix(line, "machdep.cpu.vendor") {
			c.VendorID = values[1]
		}
	}

	// Use the rated frequency of the CPU. This is a static value and does not
	// account for low power or Turbo Boost modes.
	out, err = invoke.Command(sysctl, "hw.cpufrequency")
	if err != nil {
		return ret, err
	}

	values := strings.Fields(string(out))
	mhz, err := strconv.ParseFloat(values[1], 64)
	if err != nil {
		return ret, err
	}
	c.Mhz = mhz / 1000000.0

	return append(ret, c), nil
}
