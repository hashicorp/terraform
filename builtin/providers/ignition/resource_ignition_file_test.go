package ignition

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/coreos/ignition/config/types"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestIngnitionFileReplace(t *testing.T) {
	testIgnition(t, `
		resource "ignition_file" "test" {
		    config {
		    	replace {
		    		source = "foo"
		    		verification = "sha512-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
		    	}
			}
		}
	`, func(i *types.Ignition) error {
		r := i.Config.Replace
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

func TestIngnitionFileAppend(t *testing.T) {
	testIgnition(t, `
		resource "ignition_file" "test" {
		    config {
		    	append {
		    		source = "foo"
		    		verification = "sha512-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
		    	}

		    	append {
		    		source = "foo"
		    		verification = "sha512-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
		    	}
			}
		}
	`, func(i *types.Ignition) error {
		a := i.Config.Append
		if len(a) != 2 {
			return fmt.Errorf("unable to find append config, expected 2")
		}

		if a[0].Source.String() != "foo" {
			return fmt.Errorf("config.replace.source, found %q", a[0].Source)
		}

		if a[0].Verification.Hash.Sum != "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" {
			return fmt.Errorf("config.replace.verification, found %q", a[0].Verification.Hash)
		}

		return nil
	})
}

func testIgnition(t *testing.T, input string, assert func(*types.Ignition) error) {
	check := func(s *terraform.State) error {
		got := s.RootModule().Outputs["rendered"]

		i := &types.Ignition{}
		err := json.Unmarshal([]byte(got), i)
		if err != nil {
			return err
		}

		return assert(i)
	}

	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testTemplate, input),
				Check:  check,
			},
		},
	})
}

var testTemplate = `
%s

output "rendered" {
	value = "${ignition_file.test.rendered}"
}

`
