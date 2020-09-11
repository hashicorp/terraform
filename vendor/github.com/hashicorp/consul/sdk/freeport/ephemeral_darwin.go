//+build darwin

package freeport

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
)

const ephemeralPortRangeSysctlFirst = "net.inet.ip.portrange.first"
const ephemeralPortRangeSysctlLast = "net.inet.ip.portrange.last"

var ephemeralPortRangePatt = regexp.MustCompile(`^\s*(\d+)\s+(\d+)\s*$`)

func getEphemeralPortRange() (int, int, error) {
	cmd := exec.Command("/usr/sbin/sysctl", "-n", ephemeralPortRangeSysctlFirst, ephemeralPortRangeSysctlLast)
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

	return 0, 0, fmt.Errorf("unexpected sysctl value %q for keys %q, %q", val, ephemeralPortRangeSysctlFirst, ephemeralPortRangeSysctlLast)
}
