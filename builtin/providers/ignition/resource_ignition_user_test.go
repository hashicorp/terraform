package ignition

import (
	"fmt"
	"testing"

	"github.com/coreos/ignition/config/types"
)

func TestIngnitionUser(t *testing.T) {
	testIgnition(t, `
		resource "ignition_user" "foo" {
			name = "foo"
			password_hash = "password"
			ssh_authorized_keys = ["keys"]
			uid = 42
			gecos = "gecos"
			home_dir = "home"
			no_create_home = true
			primary_group = "primary_group"
			groups = ["group"]
			no_user_group = true
			no_log_init = true
			shell = "shell"
		}

		resource "ignition_user" "qux" {
			name = "qux"
		}

		resource "ignition_config" "test" {
			users = [
				"${ignition_user.foo.id}",
				"${ignition_user.qux.id}",
			]
		}
	`, func(c *types.Config) error {
		if len(c.Passwd.Users) != 2 {
			return fmt.Errorf("users, found %d", len(c.Passwd.Users))
		}

		u := c.Passwd.Users[0]

		if u.Name != "foo" {
			return fmt.Errorf("name, found %q", u.Name)
		}

		if u.PasswordHash != "password" {
			return fmt.Errorf("password_hash, found %q", u.PasswordHash)
		}

		if len(u.SSHAuthorizedKeys) != 1 || u.SSHAuthorizedKeys[0] != "keys" {
			return fmt.Errorf("ssh_authorized_keys, found %q", u.SSHAuthorizedKeys)
		}

		if *u.Create.Uid != uint(42) {
			return fmt.Errorf("uid, found %q", *u.Create.Uid)
		}

		if u.Create.GECOS != "gecos" {
			return fmt.Errorf("gecos, found %q", u.Create.GECOS)
		}

		if u.Create.Homedir != "home" {
			return fmt.Errorf("home_dir, found %q", u.Create.Homedir)
		}

		if u.Create.NoCreateHome != true {
			return fmt.Errorf("no_create_home, found %t", u.Create.NoCreateHome)
		}

		if u.Create.PrimaryGroup != "primary_group" {
			return fmt.Errorf("primary_group, found %q", u.Create.PrimaryGroup)
		}

		if len(u.Create.Groups) != 1 || u.Create.Groups[0] != "group" {
			return fmt.Errorf("groups, found %q", u.Create.Groups)
		}

		if u.Create.NoUserGroup != true {
			return fmt.Errorf("no_create_home, found %t", u.Create.NoCreateHome)
		}

		if u.Create.NoLogInit != true {
			return fmt.Errorf("no_log_init, found %t", u.Create.NoLogInit)
		}

		if u.Create.Shell != "shell" {
			return fmt.Errorf("shell, found %q", u.Create.Shell)
		}

		u = c.Passwd.Users[1]

		if u.Name != "qux" {
			return fmt.Errorf("name, found %q", u.Name)
		}

		if u.Create != nil {
			return fmt.Errorf("create struct found")
		}

		return nil
	})
}
