// +build freebsd darwin

package net

import (
	"strings"

	"github.com/shirou/gopsutil/internal/common"
)

// Return a list of network connections opened.
func Connections(kind string) ([]ConnectionStat, error) {
	return ConnectionsPid(kind, 0)
}

// Return a list of network connections opened by a process.
func ConnectionsPid(kind string, pid int32) ([]ConnectionStat, error) {
	var ret []ConnectionStat

	args := []string{"-i"}
	switch strings.ToLower(kind) {
	default:
		fallthrough
	case "":
		fallthrough
	case "all":
		fallthrough
	case "inet":
		args = append(args, "tcp", "-i", "udp")
	case "inet4":
		args = append(args, "4")
	case "inet6":
		args = append(args, "6")
	case "tcp":
		args = append(args, "tcp")
	case "tcp4":
		args = append(args, "4tcp")
	case "tcp6":
		args = append(args, "6tcp")
	case "udp":
		args = append(args, "udp")
	case "udp4":
		args = append(args, "6udp")
	case "udp6":
		args = append(args, "6udp")
	case "unix":
		return ret, common.ErrNotImplementedError
	}

	r, err := common.CallLsof(invoke, pid, args...)
	if err != nil {
		return nil, err
	}
	for _, rr := range r {
		if strings.HasPrefix(rr, "COMMAND") {
			continue
		}
		n, err := parseNetLine(rr)
		if err != nil {

			continue
		}

		ret = append(ret, n)
	}

	return ret, nil
}
