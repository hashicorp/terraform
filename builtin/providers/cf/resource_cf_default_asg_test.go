package cloudfoundry

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/hashicorp/terraform/builtin/providers/cf/cfapi"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

const defaultRunningSecurityGroupResource = `

resource "cf_asg" "apps" {

	name = "pcf-apps"

    rule {
        destination = "192.168.100.0/24"
        protocol = "all"
    }
} 

resource "cf_asg" "services" {

	name = "pcf-services"

    rule {
        destination = "192.168.101.0/24"
        protocol = "all"
    }
} 

resource "cf_default_asg" "running" {
	name = "running"
    asgs = [ "${cf_asg.apps.id}", "${cf_asg.services.id}" ]
}
`

const defaultRunningSecurityGroupResourceUpdate = `

data "cf_asg" "public" {
    name = "public_networks"
}

resource "cf_asg" "apps" {

	name = "pcf-apps"

    rule {
        destination = "192.168.100.0/24"
        protocol = "all"
    }
}

resource "cf_asg" "services" {

	name = "pcf-services"
	
    rule {
        destination = "192.168.101.0/24"
        protocol = "all"
    }
}

resource "cf_default_asg" "running" {
	name = "running"
    asgs = [ "${data.cf_asg.public.id}", "${cf_asg.apps.id}" ]
}
`

const defaultStagingSecurityGroupResource = `

resource "cf_asg" "apps" {

	name = "pcf-apps"

    rule {
        destination = "192.168.100.0/24"
        protocol = "all"
    }
}

resource "cf_default_asg" "staging" {
	name = "staging"
    asgs = [ "${cf_asg.apps.id}" ]
}
`

func TestAccDefaultRunningAsg_normal(t *testing.T) {

	_, filename, _, _ := runtime.Caller(0)
	ut := os.Getenv("UNIT_TEST")
	if !testAccEnvironmentSet() || (len(ut) > 0 && ut != filepath.Base(filename)) {
		fmt.Printf("Skipping tests in '%s'.\n", filepath.Base(filename))
		return
	}

	ref := "cf_default_asg.running"

	resource.Test(t,
		resource.TestCase{
			PreCheck:     func() { testAccPreCheck(t) },
			Providers:    testAccProviders,
			CheckDestroy: testAccCheckDefaultRunningAsgDestroy,
			Steps: []resource.TestStep{

				resource.TestStep{
					Config: defaultRunningSecurityGroupResource,
					Check: resource.ComposeTestCheckFunc(
						checkDefaultAsgsExists(ref),
						resource.TestCheckResourceAttr(
							ref, "name", "running"),
						resource.TestCheckResourceAttr(
							ref, "asgs.#", "2"),
					),
				},
				resource.TestStep{
					Config: defaultRunningSecurityGroupResourceUpdate,
					Check: resource.ComposeTestCheckFunc(
						checkDefaultAsgsExists(ref),
						resource.TestCheckResourceAttr(
							ref, "name", "running"),
						resource.TestCheckResourceAttr(
							ref, "asgs.#", "2"),
					),
				},
			},
		})
}

func TestAccDefaultStagingAsg_normal(t *testing.T) {

	_, filename, _, _ := runtime.Caller(0)
	ut := os.Getenv("UNIT_TEST")
	if !testAccEnvironmentSet() || (len(ut) > 0 && ut != filepath.Base(filename)) {
		fmt.Printf("Skipping tests in '%s'.\n", filepath.Base(filename))
		return
	}

	ref := "cf_default_asg.staging"

	resource.Test(t,
		resource.TestCase{
			PreCheck:     func() { testAccPreCheck(t) },
			Providers:    testAccProviders,
			CheckDestroy: testAccCheckDefaultStagingAsgDestroy,
			Steps: []resource.TestStep{

				resource.TestStep{
					Config: defaultStagingSecurityGroupResource,
					Check: resource.ComposeTestCheckFunc(
						checkDefaultAsgsExists(ref),
						resource.TestCheckResourceAttr(
							ref, "name", "staging"),
						resource.TestCheckResourceAttr(
							ref, "asgs.#", "1"),
					),
				},
			},
		})
}

func checkDefaultAsgsExists(resource string) resource.TestCheckFunc {

	return func(s *terraform.State) (err error) {

		session := testAccProvider.Meta().(*cfapi.Session)

		rs, ok := s.RootModule().Resources[resource]
		if !ok {
			return fmt.Errorf("asg '%s' not found in terraform state", resource)
		}

		session.Log.DebugMessage(
			"terraform state for resource '%s': %# v",
			resource, rs)

		id := rs.Primary.ID
		attributes := rs.Primary.Attributes

		var asgs []string

		switch id {
		case "running":
			if asgs, err = session.ASGManager().Running(); err != nil {
				return
			}
		case "staging":
			if asgs, err = session.ASGManager().Staging(); err != nil {
				return
			}
		}

		if err = assertListEquals(attributes, "asgs", len(asgs),
			func(values map[string]string, i int) (match bool) {
				return values["value"] == asgs[i]
			}); err != nil {
			return
		}

		return
	}
}

func testAccCheckDefaultRunningAsgDestroy(s *terraform.State) error {

	session := testAccProvider.Meta().(*cfapi.Session)
	asgs, err := session.ASGManager().Running()
	if err != nil {
		return err
	}
	if len(asgs) > 0 {
		return fmt.Errorf("running asgs are not empty")
	}
	return nil
}

func testAccCheckDefaultStagingAsgDestroy(s *terraform.State) error {

	session := testAccProvider.Meta().(*cfapi.Session)
	asgs, err := session.ASGManager().Staging()
	if err != nil {
		return err
	}
	if len(asgs) > 0 {
		return fmt.Errorf("staging asgs are not empty")
	}
	return nil
}
