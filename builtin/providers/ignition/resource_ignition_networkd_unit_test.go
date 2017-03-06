package ignition

import (
	"fmt"
	"testing"

	"github.com/coreos/ignition/config/types"
)

func TestIngnitionNetworkdUnit(t *testing.T) {
	testIgnition(t, `
		data "ignition_networkd_unit" "foo" {
			name = "foo.link"
			content = "[Match]\nName=eth0\n\n[Network]\nAddress=10.0.1.7\n"
		}

		data "ignition_config" "test" {
			networkd = [
				"${data.ignition_networkd_unit.foo.id}",
			]
		}
	`, func(c *types.Config) error {
		if len(c.Networkd.Units) != 1 {
			return fmt.Errorf("networkd, found %d", len(c.Networkd.Units))
		}

		u := c.Networkd.Units[0]

		if u.Name != "foo.link" {
			return fmt.Errorf("name, found %q", u.Name)
		}

		if u.Contents != "[Match]\nName=eth0\n\n[Network]\nAddress=10.0.1.7\n" {
			return fmt.Errorf("content, found %q", u.Contents)
		}

		return nil
	})
}
