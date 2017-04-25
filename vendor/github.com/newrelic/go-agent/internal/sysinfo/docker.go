package sysinfo

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"runtime"
)

var (
	// ErrDockerUnsupported is returned if Docker is not supported on the
	// platform.
	ErrDockerUnsupported = errors.New("Docker unsupported on this platform")
	// ErrDockerNotFound is returned if a Docker ID is not found in
	// /proc/self/cgroup
	ErrDockerNotFound = errors.New("Docker ID not found")
)

// DockerID attempts to detect Docker.
func DockerID() (string, error) {
	if "linux" != runtime.GOOS {
		return "", ErrDockerUnsupported
	}

	f, err := os.Open("/proc/self/cgroup")
	if err != nil {
		return "", err
	}
	defer f.Close()

	return parseDockerID(f)
}

var (
	dockerIDLength   = 64
	dockerIDRegexRaw = fmt.Sprintf("^[0-9a-f]{%d}$", dockerIDLength)
	dockerIDRegex    = regexp.MustCompile(dockerIDRegexRaw)
)

func parseDockerID(r io.Reader) (string, error) {
	// Each line in the cgroup file consists of three colon delimited fields.
	//   1. hierarchy ID  - we don't care about this
	//   2. subsystems    - comma separated list of cgroup subsystem names
	//   3. control group - control group to which the process belongs
	//
	// Example
	//   5:cpuacct,cpu,cpuset:/daemons

	for scanner := bufio.NewScanner(r); scanner.Scan(); {
		line := scanner.Bytes()
		cols := bytes.SplitN(line, []byte(":"), 3)

		if len(cols) < 3 {
			continue
		}

		//  We're only interested in the cpu subsystem.
		if !isCPUCol(cols[1]) {
			continue
		}

		// We're only interested in Docker generated cgroups.
		// Reference Implementation:
		// case cpu_cgroup
		// # docker native driver w/out systemd (fs)
		// when %r{^/docker/([0-9a-f]+)$}                      then $1
		// # docker native driver with systemd
		// when %r{^/system\.slice/docker-([0-9a-f]+)\.scope$} then $1
		// # docker lxc driver
		// when %r{^/lxc/([0-9a-f]+)$}                         then $1
		//
		var id string
		if bytes.HasPrefix(cols[2], []byte("/docker/")) {
			id = string(cols[2][len("/docker/"):])
		} else if bytes.HasPrefix(cols[2], []byte("/lxc/")) {
			id = string(cols[2][len("/lxc/"):])
		} else if bytes.HasPrefix(cols[2], []byte("/system.slice/docker-")) &&
			bytes.HasSuffix(cols[2], []byte(".scope")) {
			id = string(cols[2][len("/system.slice/docker-") : len(cols[2])-len(".scope")])
		} else {
			continue
		}

		if err := validateDockerID(id); err != nil {
			// We can stop searching at this point, the CPU
			// subsystem should only occur once, and its cgroup is
			// not docker or not a format we accept.
			return "", err
		}
		return id, nil
	}

	return "", ErrDockerNotFound
}

func isCPUCol(col []byte) bool {
	// Sometimes we have multiple subsystems in one line, as in this example
	// from:
	// https://source.datanerd.us/newrelic/cross_agent_tests/blob/master/docker_container_id/docker-1.1.2-native-driver-systemd.txt
	//
	// 3:cpuacct,cpu:/system.slice/docker-67f98c9e6188f9c1818672a15dbe46237b6ee7e77f834d40d41c5fb3c2f84a2f.scope
	splitCSV := func(r rune) bool { return r == ',' }
	subsysCPU := []byte("cpu")

	for _, subsys := range bytes.FieldsFunc(col, splitCSV) {
		if bytes.Equal(subsysCPU, subsys) {
			return true
		}
	}
	return false
}

func validateDockerID(id string) error {
	if !dockerIDRegex.MatchString(id) {
		return fmt.Errorf("%s does not match %s",
			id, dockerIDRegexRaw)
	}

	return nil
}
