//+build linux

package freeport

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
)

const ephemeralPortRangeSysctlKey = "net.ipv4.ip_local_port_range"

var ephemeralPortRangePatt = regexp.MustCompile(`^\s*(\d+)\s+(\d+)\s*$`)

func getEphemeralPortRange() (int, int, error) {
	cmd := exec.Command("/sbin/sysctl", "-n", ephemeralPortRangeSysctlKey)
	out, err := cmd.Output()
	if err != nil {
		return 0, 0, err
	}

	val := string(out)

	m := ephemeralPortRangePatt.FindStringSubmatch(val)
	if m != nil {
		min, err1 := strconv.Atoi(m[1])
		max, err2 := strconv.Atoi(m[2])

		if err1 == nil && err2 == nil {
			return min, max, nil
		}
	}

	return 0, 0, fmt.Errorf("unexpected sysctl value %q for key %q", val, ephemeralPortRangeSysctlKey)
}
