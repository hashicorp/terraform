package ignition

import (
	"fmt"
	"testing"

	"github.com/coreos/ignition/config/types"
)

func TestIngnitionDisk(t *testing.T) {
	testIgnition(t, `
		data "ignition_disk" "foo" {
			device = "/foo"
			partition {
				label = "qux"
				size = 42
				start = 2048
				type_guid = "01234567-89AB-CDEF-EDCB-A98765432101"
			}
		}

		data "ignition_config" "test" {
			disks = [
				"${data.ignition_disk.foo.id}",
			]
		}
	`, func(c *types.Config) error {
		if len(c.Storage.Disks) != 1 {
			return fmt.Errorf("disks, found %d", len(c.Storage.Disks))
		}

		d := c.Storage.Disks[0]
		if d.Device != "/foo" {
			return fmt.Errorf("name, found %q", d.Device)
		}

		if len(d.Partitions) != 1 {
			return fmt.Errorf("parition, found %d", len(d.Partitions))
		}

		p := d.Partitions[0]
		if p.Label != "qux" {
			return fmt.Errorf("parition.0.label, found %q", p.Label)
		}

		if p.Size != 42 {
			return fmt.Errorf("parition.0.size, found %q", p.Size)
		}

		if p.Start != 2048 {
			return fmt.Errorf("parition.0.start, found %q", p.Start)
		}

		if p.TypeGUID != "01234567-89AB-CDEF-EDCB-A98765432101" {
			return fmt.Errorf("parition.0.type_guid, found %q", p.TypeGUID)
		}

		return nil
	})
}

func TestIngnitionDiskResource(t *testing.T) {
	testIgnition(t, `
		resource "ignition_disk" "foo" {
			device = "/foo"
			partition {
				label = "qux"
			}
		}

		data "ignition_config" "test" {
			disks = [
				"${ignition_disk.foo.id}",
			]
		}
	`, func(c *types.Config) error {
		if len(c.Storage.Disks) != 1 {
			return fmt.Errorf("disks, found %d", len(c.Storage.Disks))
		}

		return nil
	})
}
