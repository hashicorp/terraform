package ignition

import (
	"fmt"
	"testing"

	"github.com/coreos/ignition/config/types"
)

func TestIngnitiondUsers(t *testing.T) {
	testIgnition(t, `
		resource "ignition_user" "foo" {
			name = "foo"
			password_hash = "foo"
		}

		resource "ignition_user" "qux" {
			name = "qux"
			password_hash = "qux"
		}

		resource "ignition_config" "test" {
			ignition {
			    config {
			    	replace {
			    		source = "foo"
			    		verification = "sha512-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
			    	}
				}
			}

			users = [
				"${ignition_user.foo.id}",
				"${ignition_user.qux.id}",
			]
		}
	`, func(c *types.Config) error {
		r := c.Ignition.Config.Replace
		if r == nil {
			return fmt.Errorf("unable to find replace config")
		}

		if r.Source.String() != "foo" {
			return fmt.Errorf("config.replace.source, found %q", r.Source)
		}

		if r.Verification.Hash.Sum != "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" {
			return fmt.Errorf("config.replace.verification, found %q", r.Verification.Hash)
		}

		return nil
	})
}
