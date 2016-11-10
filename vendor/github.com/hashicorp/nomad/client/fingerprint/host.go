package fingerprint

import (
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"strings"

	"github.com/hashicorp/nomad/client/config"
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/shirou/gopsutil/host"
)

// HostFingerprint is used to fingerprint the host
type HostFingerprint struct {
	StaticFingerprinter
	logger *log.Logger
}

// NewHostFingerprint is used to create a Host fingerprint
func NewHostFingerprint(logger *log.Logger) Fingerprint {
	f := &HostFingerprint{logger: logger}
	return f
}

func (f *HostFingerprint) Fingerprint(cfg *config.Config, node *structs.Node) (bool, error) {
	hostInfo, err := host.Info()
	if err != nil {
		f.logger.Println("[WARN] Error retrieving host information: ", err)
		return false, err
	}

	node.Attributes["os.name"] = hostInfo.Platform
	node.Attributes["os.version"] = hostInfo.PlatformVersion

	node.Attributes["kernel.name"] = runtime.GOOS
	node.Attributes["kernel.version"] = ""

	if runtime.GOOS != "windows" {
		out, err := exec.Command("uname", "-r").Output()
		if err != nil {
			return false, fmt.Errorf("Failed to run uname: %s", err)
		}
		node.Attributes["kernel.version"] = strings.Trim(string(out), "\n")
	}

	node.Attributes["unique.hostname"] = hostInfo.Hostname

	return true, nil
}
