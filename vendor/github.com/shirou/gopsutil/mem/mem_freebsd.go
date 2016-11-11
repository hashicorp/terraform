// +build freebsd

package mem

import (
	"errors"
	"os/exec"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/internal/common"
)

func VirtualMemory() (*VirtualMemoryStat, error) {
	pageSize, err := common.DoSysctrl("vm.stats.vm.v_page_size")
	if err != nil {
		return nil, err
	}
	p, err := strconv.ParseUint(pageSize[0], 10, 64)
	if err != nil {
		return nil, err
	}

	pageCount, err := common.DoSysctrl("vm.stats.vm.v_page_count")
	if err != nil {
		return nil, err
	}
	free, err := common.DoSysctrl("vm.stats.vm.v_free_count")
	if err != nil {
		return nil, err
	}
	active, err := common.DoSysctrl("vm.stats.vm.v_active_count")
	if err != nil {
		return nil, err
	}
	inactive, err := common.DoSysctrl("vm.stats.vm.v_inactive_count")
	if err != nil {
		return nil, err
	}
	cache, err := common.DoSysctrl("vm.stats.vm.v_cache_count")
	if err != nil {
		return nil, err
	}
	buffer, err := common.DoSysctrl("vfs.bufspace")
	if err != nil {
		return nil, err
	}
	wired, err := common.DoSysctrl("vm.stats.vm.v_wire_count")
	if err != nil {
		return nil, err
	}

	parsed := make([]uint64, 0, 7)
	vv := []string{
		pageCount[0],
		free[0],
		active[0],
		inactive[0],
		cache[0],
		buffer[0],
		wired[0],
	}
	for _, target := range vv {
		t, err := strconv.ParseUint(target, 10, 64)
		if err != nil {
			return nil, err
		}
		parsed = append(parsed, t)
	}

	ret := &VirtualMemoryStat{
		Total:    parsed[0] * p,
		Free:     parsed[1] * p,
		Active:   parsed[2] * p,
		Inactive: parsed[3] * p,
		Cached:   parsed[4] * p,
		Buffers:  parsed[5],
		Wired:    parsed[6] * p,
	}

	ret.Available = ret.Inactive + ret.Cached + ret.Free
	ret.Used = ret.Total - ret.Available
	ret.UsedPercent = float64(ret.Used) / float64(ret.Total) * 100.0

	return ret, nil
}

// Return swapinfo
// FreeBSD can have multiple swap devices. but use only first device
func SwapMemory() (*SwapMemoryStat, error) {
	swapinfo, err := exec.LookPath("swapinfo")
	if err != nil {
		return nil, err
	}

	out, err := invoke.Command(swapinfo)
	if err != nil {
		return nil, err
	}
	for _, line := range strings.Split(string(out), "\n") {
		values := strings.Fields(line)
		// skip title line
		if len(values) == 0 || values[0] == "Device" {
			continue
		}

		u := strings.Replace(values[4], "%", "", 1)
		total_v, err := strconv.ParseUint(values[1], 10, 64)
		if err != nil {
			return nil, err
		}
		used_v, err := strconv.ParseUint(values[2], 10, 64)
		if err != nil {
			return nil, err
		}
		free_v, err := strconv.ParseUint(values[3], 10, 64)
		if err != nil {
			return nil, err
		}
		up_v, err := strconv.ParseFloat(u, 64)
		if err != nil {
			return nil, err
		}

		return &SwapMemoryStat{
			Total:       total_v,
			Used:        used_v,
			Free:        free_v,
			UsedPercent: up_v,
		}, nil
	}

	return nil, errors.New("no swap devices found")
}
