package driver

import (
	"github.com/hashicorp/nomad/client/config"
	"github.com/hashicorp/nomad/nomad/structs"
	"golang.org/x/sys/unix"
)

func (d *ExecDriver) Fingerprint(cfg *config.Config, node *structs.Node) (bool, error) {
	// Get the current status so that we can log any debug messages only if the
	// state changes
	_, currentlyEnabled := node.Attributes[execDriverAttr]

	// Only enable if cgroups are available and we are root
	if _, ok := node.Attributes["unique.cgroup.mountpoint"]; !ok {
		if currentlyEnabled {
			d.logger.Printf("[DEBUG] driver.exec: cgroups unavailable, disabling")
		}
		delete(node.Attributes, execDriverAttr)
		return false, nil
	} else if unix.Geteuid() != 0 {
		if currentlyEnabled {
			d.logger.Printf("[DEBUG] driver.exec: must run as root user, disabling")
		}
		delete(node.Attributes, execDriverAttr)
		return false, nil
	}

	if !currentlyEnabled {
		d.logger.Printf("[DEBUG] driver.exec: exec driver is enabled")
	}
	node.Attributes[execDriverAttr] = "1"
	return true, nil
}
