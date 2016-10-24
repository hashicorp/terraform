// +build darwin

package mem

import (
	"encoding/binary"
	"strconv"
	"strings"
	"syscall"

	"github.com/shirou/gopsutil/internal/common"
)

func getHwMemsize() (uint64, error) {
	totalString, err := syscall.Sysctl("hw.memsize")
	if err != nil {
		return 0, err
	}

	// syscall.sysctl() helpfully assumes the result is a null-terminated string and
	// removes the last byte of the result if it's 0 :/
	totalString += "\x00"

	total := uint64(binary.LittleEndian.Uint64([]byte(totalString)))

	return total, nil
}

// SwapMemory returns swapinfo.
func SwapMemory() (*SwapMemoryStat, error) {
	var ret *SwapMemoryStat

	swapUsage, err := common.DoSysctrl("vm.swapusage")
	if err != nil {
		return ret, err
	}

	total := strings.Replace(swapUsage[2], "M", "", 1)
	used := strings.Replace(swapUsage[5], "M", "", 1)
	free := strings.Replace(swapUsage[8], "M", "", 1)

	total_v, err := strconv.ParseFloat(total, 64)
	if err != nil {
		return nil, err
	}
	used_v, err := strconv.ParseFloat(used, 64)
	if err != nil {
		return nil, err
	}
	free_v, err := strconv.ParseFloat(free, 64)
	if err != nil {
		return nil, err
	}

	u := float64(0)
	if total_v != 0 {
		u = ((total_v - free_v) / total_v) * 100.0
	}

	// vm.swapusage shows "M", multiply 1000
	ret = &SwapMemoryStat{
		Total:       uint64(total_v * 1000),
		Used:        uint64(used_v * 1000),
		Free:        uint64(free_v * 1000),
		UsedPercent: u,
	}

	return ret, nil
}
