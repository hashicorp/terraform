package ignition

import (
	"fmt"
	"testing"

	"github.com/coreos/ignition/config/types"
)

func TestIngnitionFile(t *testing.T) {
	testIgnition(t, `
		data "ignition_file" "foo" {
			filesystem = "foo"
			path = "/foo"
			content {
				content = "foo"
			}
			mode = 420
			uid = 42
			gid = 84
		}

		data "ignition_file" "qux" {
			filesystem = "qux"
			path = "/qux"
			source {
				source = "qux"
				compression = "gzip"
				verification = "sha512-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
			}
		}

		data "ignition_config" "test" {
			files = [
				"${data.ignition_file.foo.id}",
				"${data.ignition_file.qux.id}",
			]
		}
	`, func(c *types.Config) error {
		if len(c.Storage.Files) != 2 {
			return fmt.Errorf("arrays, found %d", len(c.Storage.Arrays))
		}

		f := c.Storage.Files[0]
		if f.Filesystem != "foo" {
			return fmt.Errorf("filesystem, found %q", f.Filesystem)
		}

		if f.Path != "/foo" {
			return fmt.Errorf("path, found %q", f.Path)
		}

		if f.Contents.Source.String() != "data:text/plain;charset=utf-8;base64,Zm9v" {
			return fmt.Errorf("contents.source, found %q", f.Contents.Source)
		}

		if f.Mode != types.FileMode(420) {
			return fmt.Errorf("mode, found %q", f.Mode)
		}

		if f.User.Id != 42 {
			return fmt.Errorf("uid, found %q", f.User.Id)
		}

		if f.Group.Id != 84 {
			return fmt.Errorf("gid, found %q", f.Group.Id)
		}

		f = c.Storage.Files[1]
		if f.Filesystem != "qux" {
			return fmt.Errorf("filesystem, found %q", f.Filesystem)
		}

		if f.Path != "/qux" {
			return fmt.Errorf("path, found %q", f.Path)
		}

		if f.Contents.Source.String() != "qux" {
			return fmt.Errorf("contents.source, found %q", f.Contents.Source)
		}

		if f.Contents.Compression != "gzip" {
			return fmt.Errorf("contents.compression, found %q", f.Contents.Compression)
		}

		if f.Contents.Verification.Hash.Sum != "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" {
			return fmt.Errorf("config.replace.verification, found %q", f.Contents.Verification.Hash)
		}

		return nil
	})
}
