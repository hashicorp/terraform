//+build darwin dragonfly freebsd netbsd openbsd solaris windows

package driver

import (
	"github.com/hashicorp/nomad/client/config"
	"github.com/hashicorp/nomad/nomad/structs"
)

func (d *ExecDriver) Fingerprint(cfg *config.Config, node *structs.Node) (bool, error) {
	return false, nil
}
