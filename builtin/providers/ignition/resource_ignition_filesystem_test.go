package ignition

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/coreos/ignition/config/types"
)

func TestIngnitionFilesystem(t *testing.T) {
	testIgnition(t, `
		data "ignition_filesystem" "foo" {
			name = "foo"
			path = "/foo"
		}

		data "ignition_filesystem" "qux" {
			name = "qux"
			mount {
				device = "/qux"
				format = "ext4"
			}
		}

		data "ignition_filesystem" "baz" {
			name = "baz"
			mount {
				device = "/baz"
				format = "ext4"
				create = true
			}
		}

		data "ignition_filesystem" "bar" {
			name = "bar"
			mount {
				device = "/bar"
				format = "ext4"
				create = true
				force = true
				options = ["rw"]
			}
		}

		data "ignition_config" "test" {
			filesystems = [
				"${data.ignition_filesystem.foo.id}",
				"${data.ignition_filesystem.qux.id}",
				"${data.ignition_filesystem.baz.id}",
				"${data.ignition_filesystem.bar.id}",
			]
		}
	`, func(c *types.Config) error {
		if len(c.Storage.Filesystems) != 4 {
			return fmt.Errorf("disks, found %d", len(c.Storage.Filesystems))
		}

		f := c.Storage.Filesystems[0]
		if f.Name != "foo" {
			return fmt.Errorf("name, found %q", f.Name)
		}

		if f.Mount != nil {
			return fmt.Errorf("mount, found %q", f.Mount.Device)
		}

		if string(*f.Path) != "/foo" {
			return fmt.Errorf("path, found %q", f.Path)
		}

		f = c.Storage.Filesystems[1]
		if f.Name != "qux" {
			return fmt.Errorf("name, found %q", f.Name)
		}

		if f.Mount.Device != "/qux" {
			return fmt.Errorf("mount.0.device, found %q", f.Mount.Device)
		}

		if f.Mount.Format != "ext4" {
			return fmt.Errorf("mount.0.format, found %q", f.Mount.Format)
		}

		if f.Mount.Create != nil {
			return fmt.Errorf("mount, create was found %#v", f.Mount.Create)
		}

		f = c.Storage.Filesystems[2]
		if f.Name != "baz" {
			return fmt.Errorf("name, found %q", f.Name)
		}

		if f.Mount.Device != "/baz" {
			return fmt.Errorf("mount.0.device, found %q", f.Mount.Device)
		}

		if f.Mount.Format != "ext4" {
			return fmt.Errorf("mount.0.format, found %q", f.Mount.Format)
		}

		if f.Mount.Create.Force != false {
			return fmt.Errorf("mount.0.force, found %t", f.Mount.Create.Force)
		}

		f = c.Storage.Filesystems[3]
		if f.Name != "bar" {
			return fmt.Errorf("name, found %q", f.Name)
		}

		if f.Mount.Device != "/bar" {
			return fmt.Errorf("mount.0.device, found %q", f.Mount.Device)
		}

		if f.Mount.Format != "ext4" {
			return fmt.Errorf("mount.0.format, found %q", f.Mount.Format)
		}

		if f.Mount.Create.Force != true {
			return fmt.Errorf("mount.0.force, found %t", f.Mount.Create.Force)
		}

		if len(f.Mount.Create.Options) != 1 || f.Mount.Create.Options[0] != "rw" {
			return fmt.Errorf("mount.0.options, found %q", f.Mount.Create.Options)
		}

		return nil
	})
}

func TestIngnitionFilesystemMissingCreate(t *testing.T) {
	testIgnitionError(t, `
		data "ignition_filesystem" "bar" {
			name = "bar"
			mount {
				device = "/bar"
				format = "ext4"
				force = true
			}
		}

		data "ignition_config" "test" {
			filesystems = [
				"${data.ignition_filesystem.bar.id}",
			]
		}
	`, regexp.MustCompile("create should be true when force or options is used"))
}
