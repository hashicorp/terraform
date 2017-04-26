package ignition

import (
	"fmt"
	"testing"

	"github.com/coreos/ignition/config/types"
)

func TestIngnitionSystemdUnit(t *testing.T) {
	testIgnition(t, `
		data "ignition_systemd_unit" "foo" {
			name = "foo.service"
			content = "[Match]\nName=eth0\n\n[Network]\nAddress=10.0.1.7\n"
			enable = false
			mask = true

			dropin {
				name = "foo.conf"
				content = "[Match]\nName=eth0\n\n[Network]\nAddress=10.0.1.7\n"
			}
		}

		data "ignition_config" "test" {
			systemd = [
				"${data.ignition_systemd_unit.foo.id}",
			]
		}
	`, func(c *types.Config) error {
		if len(c.Systemd.Units) != 1 {
			return fmt.Errorf("systemd, found %d", len(c.Systemd.Units))
		}

		u := c.Systemd.Units[0]

		if u.Name != "foo.service" {
			return fmt.Errorf("name, found %q", u.Name)
		}

		if u.Contents != "[Match]\nName=eth0\n\n[Network]\nAddress=10.0.1.7\n" {
			return fmt.Errorf("content, found %q", u.Contents)
		}

		if u.Mask != true {
			return fmt.Errorf("mask, found %t", u.Mask)
		}

		if u.Enable != false {
			return fmt.Errorf("enable, found %t", u.Enable)
		}

		if len(u.DropIns) != 1 {
			return fmt.Errorf("dropins, found %q", u.DropIns)
		}

		return nil
	})
}

func TestIngnitionSystemdUnitEmptyContentWithDropIn(t *testing.T) {
	testIgnition(t, `
		data "ignition_systemd_unit" "foo" {
			name = "foo.service"
			dropin {
				name = "foo.conf"
				content = "[Match]\nName=eth0\n\n[Network]\nAddress=10.0.1.7\n"
			}
		}

		data "ignition_config" "test" {
			systemd = [
				"${data.ignition_systemd_unit.foo.id}",
			]
		}
	`, func(c *types.Config) error {
		if len(c.Systemd.Units) != 1 {
			return fmt.Errorf("systemd, found %d", len(c.Systemd.Units))
		}

		u := c.Systemd.Units[0]

		if u.Name != "foo.service" {
			return fmt.Errorf("name, found %q", u.Name)
		}

		if u.Contents != "" {
			return fmt.Errorf("content, found %q", u.Contents)
		}

		if len(u.DropIns) != 1 {
			return fmt.Errorf("dropins, found %q", u.DropIns)
		}

		return nil
	})
}

// #11325
func TestIgnitionSystemdUnit_emptyContent(t *testing.T) {
	testIgnition(t, `
		data "ignition_systemd_unit" "foo" {
			name = "foo.service"
			enable = true
		}

		data "ignition_config" "test" {
			systemd = [
				"${data.ignition_systemd_unit.foo.id}",
			]
		}
	`, func(c *types.Config) error {
		if len(c.Systemd.Units) != 1 {
			return fmt.Errorf("systemd, found %d", len(c.Systemd.Units))
		}

		u := c.Systemd.Units[0]
		if u.Name != "foo.service" {
			return fmt.Errorf("name, expected 'foo.service', found %q", u.Name)
		}
		if u.Contents != "" {
			return fmt.Errorf("expected empty content, found %q", u.Contents)
		}
		if len(u.DropIns) != 0 {
			return fmt.Errorf("expected 0 dropins, found %q", u.DropIns)
		}
		return nil
	})
}
