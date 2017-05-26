package sysinfo

import "os"

// PhysicalMemoryBytes returns the total amount of host memory.
func PhysicalMemoryBytes() (uint64, error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, err
	}
	defer f.Close()

	return parseProcMeminfo(f)
}
