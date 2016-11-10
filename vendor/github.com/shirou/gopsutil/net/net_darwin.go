// +build darwin

package net

import (
	"errors"
	"os/exec"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/internal/common"
)

// example of netstat -idbn output on yosemite
// Name  Mtu   Network       Address            Ipkts Ierrs     Ibytes    Opkts Oerrs     Obytes  Coll Drop
// lo0   16384 <Link#1>                        869107     0  169411755   869107     0  169411755     0   0
// lo0   16384 ::1/128     ::1                 869107     -  169411755   869107     -  169411755     -   -
// lo0   16384 127           127.0.0.1         869107     -  169411755   869107     -  169411755     -   -
func IOCounters(pernic bool) ([]IOCountersStat, error) {
	netstat, err := exec.LookPath("/usr/sbin/netstat")
	if err != nil {
		return nil, err
	}
	out, err := invoke.Command(netstat, "-ibdnW")
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(out), "\n")
	ret := make([]IOCountersStat, 0, len(lines)-1)
	exists := make([]string, 0, len(ret))

	for _, line := range lines {
		values := strings.Fields(line)
		if len(values) < 1 || values[0] == "Name" {
			// skip first line
			continue
		}
		if common.StringsHas(exists, values[0]) {
			// skip if already get
			continue
		}
		exists = append(exists, values[0])

		base := 1
		// sometimes Address is ommitted
		if len(values) < 11 {
			base = 0
		}

		parsed := make([]uint64, 0, 7)
		vv := []string{
			values[base+3], // Ipkts == PacketsRecv
			values[base+4], // Ierrs == Errin
			values[base+5], // Ibytes == BytesRecv
			values[base+6], // Opkts == PacketsSent
			values[base+7], // Oerrs == Errout
			values[base+8], // Obytes == BytesSent
		}
		if len(values) == 12 {
			vv = append(vv, values[base+10])
		}

		for _, target := range vv {
			if target == "-" {
				parsed = append(parsed, 0)
				continue
			}

			t, err := strconv.ParseUint(target, 10, 64)
			if err != nil {
				return nil, err
			}
			parsed = append(parsed, t)
		}

		n := IOCountersStat{
			Name:        values[0],
			PacketsRecv: parsed[0],
			Errin:       parsed[1],
			BytesRecv:   parsed[2],
			PacketsSent: parsed[3],
			Errout:      parsed[4],
			BytesSent:   parsed[5],
		}
		if len(parsed) == 7 {
			n.Dropout = parsed[6]
		}
		ret = append(ret, n)
	}

	if pernic == false {
		return getIOCountersAll(ret)
	}

	return ret, nil
}

// NetIOCountersByFile is an method which is added just a compatibility for linux.
func IOCountersByFile(pernic bool, filename string) ([]IOCountersStat, error) {
	return IOCounters(pernic)
}

func FilterCounters() ([]FilterStat, error) {
	return nil, errors.New("NetFilterCounters not implemented for darwin")
}

// NetProtoCounters returns network statistics for the entire system
// If protocols is empty then all protocols are returned, otherwise
// just the protocols in the list are returned.
// Not Implemented for Darwin
func ProtoCounters(protocols []string) ([]ProtoCountersStat, error) {
	return nil, errors.New("NetProtoCounters not implemented for darwin")
}
