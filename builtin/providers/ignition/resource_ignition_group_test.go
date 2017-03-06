package ignition

import (
	"fmt"
	"testing"

	"github.com/coreos/ignition/config/types"
)

func TestIngnitionGroup(t *testing.T) {
	testIgnition(t, `
		data "ignition_group" "foo" {
			name = "foo"
			password_hash = "password"
			gid = 42
		}

		data "ignition_group" "qux" {
			name = "qux"
		}

		data "ignition_config" "test" {
			groups = [
				"${data.ignition_group.foo.id}",
				"${data.ignition_group.qux.id}",
			]
		}
	`, func(c *types.Config) error {
		if len(c.Passwd.Groups) != 2 {
			return fmt.Errorf("groups, found %d", len(c.Passwd.Groups))
		}

		g := c.Passwd.Groups[0]

		if g.Name != "foo" {
			return fmt.Errorf("name, found %q", g.Name)
		}

		if g.PasswordHash != "password" {
			return fmt.Errorf("password_hash, found %q", g.PasswordHash)
		}

		if g.Gid == nil || *g.Gid != uint(42) {
			return fmt.Errorf("gid, found %q", *g.Gid)
		}

		g = c.Passwd.Groups[1]

		if g.Name != "qux" {
			return fmt.Errorf("name, found %q", g.Name)
		}

		if g.Gid != nil {
			return fmt.Errorf("uid, found %d", *g.Gid)
		}

		return nil
	})
}
