package ignition

import (
	"fmt"
	"testing"

	"github.com/coreos/ignition/config/types"
)

func TestIngnitionUser(t *testing.T) {
	testIgnition(t, `
		data "ignition_user" "foo" {
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

		data "ignition_user" "qux" {
			name = "qux"
		}

		data "ignition_config" "test" {
			users = [
				"${data.ignition_user.foo.id}",
				"${data.ignition_user.qux.id}",
			]
		}
	`, func(c *types.Config) error {
		if len(c.Passwd.Users) != 2 {
			return fmt.Errorf("Lenght of field Users didn't match. Expected: %d, Given: %d", 2, len(c.Passwd.Users))
		}

		u := c.Passwd.Users[0]

		if u.Name != "foo" {
			return fmt.Errorf("Field Name didn't match. Expected: %s, Given: %s", "foo", u.Name)
		}

		if u.PasswordHash != "password" {
			return fmt.Errorf("Field PasswordHash didn't match. Expected: %s, Given: %s", "password", u.PasswordHash)
		}

		if len(u.SSHAuthorizedKeys) != 1 {
			return fmt.Errorf("Lenght of field SSHAuthorizedKeys didn't match. Expected: %d, Given: %d", 1, len(u.SSHAuthorizedKeys))
		}

		if u.SSHAuthorizedKeys[0] != "keys" {
			return fmt.Errorf("Field SSHAuthorizedKeys didn't match. Expected: %s, Given: %s", "keys", u.SSHAuthorizedKeys[0])
		}

		if *u.Create.Uid != uint(42) {
			return fmt.Errorf("Field Uid didn't match. Expected: %d, Given: %d", uint(42), u.Create.Uid)
		}

		if u.Create.GECOS != "gecos" {
			return fmt.Errorf("Field GECOS didn't match. Expected: %s, Given: %s", "gecos", u.Create.GECOS)
		}

		if u.Create.Homedir != "home" {
			return fmt.Errorf("Field Homedir didn't match. Expected: %s, Given: %s", "home", u.Create.Homedir)
		}

		if u.Create.NoCreateHome != true {
			return fmt.Errorf("Field NoCreateHome didn't match. Expected: %t, Given: %t", true, u.Create.NoCreateHome)
		}

		if u.Create.PrimaryGroup != "primary_group" {
			return fmt.Errorf("Field PrimaryGroup didn't match. Expected: %s, Given: %s", "primary_group", u.Create.PrimaryGroup)
		}

		if len(u.Create.Groups) != 1 {
			return fmt.Errorf("Lenght of field Groups didn't match. Expected: %d, Given: %d", 1, len(u.Create.Groups))
		}

		if u.Create.Groups[0] != "group" {
			return fmt.Errorf("Field Groups didn't match. Expected: %s, Given: %s", "group", u.Create.Groups[0])
		}

		if u.Create.NoUserGroup != true {
			return fmt.Errorf("Field NoUserGroup didn't match. Expected: %t, Given: %t", true, u.Create.NoUserGroup)
		}

		if u.Create.NoLogInit != true {
			return fmt.Errorf("Field NoLogInit didn't match. Expected: %t, Given: %t", true, u.Create.NoLogInit)
		}

		if u.Create.Shell != "shell" {
			return fmt.Errorf("Field Shell didn't match. Expected: %s, Given: %s", "shell", u.Create.Shell)
		}

		u = c.Passwd.Users[1]

		if u.Name != "qux" {
			return fmt.Errorf("Field Name didn't match. Expected: %s, Given: %s", "qux", u.Name)
		}

		if u.Create != nil {
			return fmt.Errorf("Field Create didn't match. Expected: %v, Given: %v", nil, u.Create)
		}

		return nil
	})
}
