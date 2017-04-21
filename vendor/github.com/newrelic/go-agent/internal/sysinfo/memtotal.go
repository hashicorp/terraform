package sysinfo

import (
	"bufio"
	"errors"
	"io"
	"regexp"
	"strconv"
)

// BytesToMebibytes converts bytes into mebibytes.
func BytesToMebibytes(bts uint64) uint64 {
	return bts / ((uint64)(1024 * 1024))
}

var (
	meminfoRe           = regexp.MustCompile(`^MemTotal:\s+([0-9]+)\s+[kK]B$`)
	errMemTotalNotFound = errors.New("supported MemTotal not found in /proc/meminfo")
)

// parseProcMeminfo is used to parse Linux's "/proc/meminfo".  It is located
// here so that the relevant cross agent tests will be run on all platforms.
func parseProcMeminfo(f io.Reader) (uint64, error) {
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if m := meminfoRe.FindSubmatch(scanner.Bytes()); m != nil {
			kb, err := strconv.ParseUint(string(m[1]), 10, 64)
			if err != nil {
				return 0, err
			}
			return kb * 1024, nil
		}
	}

	err := scanner.Err()
	if err == nil {
		err = errMemTotalNotFound
	}
	return 0, err
}
