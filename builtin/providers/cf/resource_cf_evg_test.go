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

const evgRunningResource = `

resource "cf_evg" "running" {

	name = "running"

    variables = {
        name1 = "value1"
        name2 = "value2"
        name3 = "value3"
        name4 = "value4"
    }
}
`

const evgRunningResourceUpdated = `

resource "cf_evg" "running" {

	name = "running"

    variables = {
        name1 = "value1"
        name2 = "value2"
        name3 = "valueC"
        name4 = "valueD"
        name5 = "valueE"
    }
}
`

const evgStagingResource = `
resource "cf_evg" "staging" {

	name = "staging"

    variables = {
        name3 = "value3"
        name4 = "value4"
        name5 = "value5"
    }    
}
`

const evgStagingResourceUpdated = `
resource "cf_evg" "staging" {

	name = "staging"

    variables = {
        name4 = "value4"
        name5 = "valueE"
    }    
}
`

func TestAccRunningEvg_normal(t *testing.T) {

	_, filename, _, _ := runtime.Caller(0)
	ut := os.Getenv("UNIT_TEST")
	if !testAccEnvironmentSet() || (len(ut) > 0 && ut != filepath.Base(filename)) {
		fmt.Printf("Skipping tests in '%s'.\n", filepath.Base(filename))
		return
	}

	ref := "cf_evg.running"
	name := "running"

	resource.Test(t,
		resource.TestCase{
			PreCheck:     func() { testAccPreCheck(t) },
			Providers:    testAccProviders,
			CheckDestroy: testAccCheckEvgDestroy(name),
			Steps: []resource.TestStep{

				resource.TestStep{
					Config: evgRunningResource,
					Check: resource.ComposeTestCheckFunc(
						checkEvgExists(ref),
						resource.TestCheckResourceAttr(
							ref, "name", "running"),
						resource.TestCheckResourceAttr(
							ref, "variables.%", "4"),
						resource.TestCheckResourceAttr(
							ref, "variables.name1", "value1"),
						resource.TestCheckResourceAttr(
							ref, "variables.name2", "value2"),
						resource.TestCheckResourceAttr(
							ref, "variables.name3", "value3"),
						resource.TestCheckResourceAttr(
							ref, "variables.name4", "value4"),
					),
				},
				resource.TestStep{
					Config: evgRunningResourceUpdated,
					Check: resource.ComposeTestCheckFunc(
						checkEvgExists(ref),
						resource.TestCheckResourceAttr(
							ref, "name", "running"),
						resource.TestCheckResourceAttr(
							ref, "variables.%", "5"),
						resource.TestCheckResourceAttr(
							ref, "variables.name1", "value1"),
						resource.TestCheckResourceAttr(
							ref, "variables.name2", "value2"),
						resource.TestCheckResourceAttr(
							ref, "variables.name3", "valueC"),
						resource.TestCheckResourceAttr(
							ref, "variables.name4", "valueD"),
						resource.TestCheckResourceAttr(
							ref, "variables.name5", "valueE"),
					),
				},
			},
		})
}

func TestAccStagingEvg_normal(t *testing.T) {

	_, filename, _, _ := runtime.Caller(0)
	ut := os.Getenv("UNIT_TEST")
	if !testAccEnvironmentSet() || (len(ut) > 0 && ut != filepath.Base(filename)) {
		fmt.Printf("Skipping tests in '%s'.\n", filepath.Base(filename))
		return
	}

	ref := "cf_evg.staging"
	name := "staging"

	resource.Test(t,
		resource.TestCase{
			PreCheck:     func() { testAccPreCheck(t) },
			Providers:    testAccProviders,
			CheckDestroy: testAccCheckEvgDestroy(name),
			Steps: []resource.TestStep{

				resource.TestStep{
					Config: evgStagingResource,
					Check: resource.ComposeTestCheckFunc(
						checkEvgExists(ref),
						resource.TestCheckResourceAttr(
							ref, "name", "staging"),
						resource.TestCheckResourceAttr(
							ref, "variables.%", "3"),
						resource.TestCheckResourceAttr(
							ref, "variables.name3", "value3"),
						resource.TestCheckResourceAttr(
							ref, "variables.name4", "value4"),
						resource.TestCheckResourceAttr(
							ref, "variables.name5", "value5"),
					),
				},
				resource.TestStep{
					Config: evgStagingResourceUpdated,
					Check: resource.ComposeTestCheckFunc(
						checkEvgExists(ref),
						resource.TestCheckResourceAttr(
							ref, "name", "staging"),
						resource.TestCheckResourceAttr(
							ref, "variables.%", "2"),
						resource.TestCheckResourceAttr(
							ref, "variables.name4", "value4"),
						resource.TestCheckResourceAttr(
							ref, "variables.name5", "valueE"),
					),
				},
			},
		})
}

func checkEvgExists(resource string) resource.TestCheckFunc {

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

		session.Log.DebugMessage(
			"terraform state for attributes '%s': %# v",
			resource, attributes)

		variables, err := session.EVGManager().GetEVG(id)
		if err := asserMapEquals("variables", attributes, variables); err != nil {
			return err
		}
		return nil
	}
}

func testAccCheckEvgDestroy(name string) resource.TestCheckFunc {

	return func(s *terraform.State) (err error) {
		session := testAccProvider.Meta().(*cfapi.Session)
		variables, err := session.EVGManager().GetEVG(name)
		if err != nil {
			return err
		}
		if len(variables) > 0 {
			return fmt.Errorf("%s variables are not empty", name)
		}
		return nil
	}
}
