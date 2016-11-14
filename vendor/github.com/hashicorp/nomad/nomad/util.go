package nomad

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/hashicorp/serf/serf"
)

// ensurePath is used to make sure a path exists
func ensurePath(path string, dir bool) error {
	if !dir {
		path = filepath.Dir(path)
	}
	return os.MkdirAll(path, 0755)
}

// RuntimeStats is used to return various runtime information
func RuntimeStats() map[string]string {
	return map[string]string{
		"kernel.name": runtime.GOOS,
		"arch":        runtime.GOARCH,
		"version":     runtime.Version(),
		"max_procs":   strconv.FormatInt(int64(runtime.GOMAXPROCS(0)), 10),
		"goroutines":  strconv.FormatInt(int64(runtime.NumGoroutine()), 10),
		"cpu_count":   strconv.FormatInt(int64(runtime.NumCPU()), 10),
	}
}

// serverParts is used to return the parts of a server role
type serverParts struct {
	Name         string
	Region       string
	Datacenter   string
	Port         int
	Bootstrap    bool
	Expect       int
	MajorVersion int
	MinorVersion int
	Addr         net.Addr
}

func (s *serverParts) String() string {
	return fmt.Sprintf("%s (Addr: %s) (DC: %s)",
		s.Name, s.Addr, s.Datacenter)
}

// Returns if a member is a Nomad server. Returns a boolean,
// and a struct with the various important components
func isNomadServer(m serf.Member) (bool, *serverParts) {
	if m.Tags["role"] != "nomad" {
		return false, nil
	}

	region := m.Tags["region"]
	datacenter := m.Tags["dc"]
	_, bootstrap := m.Tags["bootstrap"]

	expect := 0
	expect_str, ok := m.Tags["expect"]
	var err error
	if ok {
		expect, err = strconv.Atoi(expect_str)
		if err != nil {
			return false, nil
		}
	}

	port_str := m.Tags["port"]
	port, err := strconv.Atoi(port_str)
	if err != nil {
		return false, nil
	}

	// The "vsn" tag was Version, which is now the MajorVersion number.
	majorVersionStr := m.Tags["vsn"]
	majorVersion, err := strconv.Atoi(majorVersionStr)
	if err != nil {
		return false, nil
	}

	// To keep some semblance of convention, "mvn" is now the "Minor
	// Version Number."
	minorVersionStr := m.Tags["mvn"]
	minorVersion, err := strconv.Atoi(minorVersionStr)
	if err != nil {
		minorVersion = 0
	}

	addr := &net.TCPAddr{IP: m.Addr, Port: port}
	parts := &serverParts{
		Name:         m.Name,
		Region:       region,
		Datacenter:   datacenter,
		Port:         port,
		Bootstrap:    bootstrap,
		Expect:       expect,
		Addr:         addr,
		MajorVersion: majorVersion,
		MinorVersion: minorVersion,
	}
	return true, parts
}

// shuffleStrings randomly shuffles the list of strings
func shuffleStrings(list []string) {
	for i := range list {
		j := rand.Intn(i + 1)
		list[i], list[j] = list[j], list[i]
	}
}

// maxUint64 returns the maximum value
func maxUint64(a, b uint64) uint64 {
	if a >= b {
		return a
	}
	return b
}
