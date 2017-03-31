package cloudfoundry

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	"code.cloudfoundry.org/cli/cf/errors"

	"github.com/hashicorp/terraform/builtin/providers/cf/cfapi"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

const securityGroup = `
resource "cf_asg" "rmq" {

	name = "rmq-dev"
	
    rule {
        protocol = "tcp"
        destination = "192.168.1.100"
        ports = "5672,5671,1883,8883,61613,61614"
		log = true
    }
    rule {
        protocol = "udp"
        destination = "192.168.1.101"
        ports = "5674,5673"
    }
}
`

const securityGroupUpdate = `
resource "cf_asg" "rmq" {

	name = "rmq-dev"
	
    rule {
        protocol = "tcp"
        destination = "192.168.1.100"
        ports = "61613,61614"
    }
    rule {
        protocol = "tcp"
        destination = "192.168.1.0/24"
        ports = "61613,61614"
    }
    rule {
        protocol = "all",
        destination = "0.0.0.0/0"
		log = true
    }
}
`

func TestAccAsg_normal(t *testing.T) {

	_, filename, _, _ := runtime.Caller(0)
	ut := os.Getenv("UNIT_TEST")
	if !testAccEnvironmentSet() || (len(ut) > 0 && ut != filepath.Base(filename)) {
		fmt.Printf("Skipping tests in '%s'.\n", filepath.Base(filename))
		return
	}

	ref := "cf_asg.rmq"
	asgname := "rmq-dev"

	resource.Test(t,
		resource.TestCase{
			PreCheck:     func() { testAccPreCheck(t) },
			Providers:    testAccProviders,
			CheckDestroy: testAccCheckASGDestroy(asgname),
			Steps: []resource.TestStep{

				resource.TestStep{
					Config: securityGroup,
					Check: resource.ComposeTestCheckFunc(
						testAccCheckASGExists(ref),
						resource.TestCheckResourceAttr(
							ref, "name", asgname),
						resource.TestCheckResourceAttr(
							ref, "rule.#", "2"),
						resource.TestCheckResourceAttr(
							ref, "rule.0.protocol", "tcp"),
						resource.TestCheckResourceAttr(
							ref, "rule.0.destination", "192.168.1.100"),
						resource.TestCheckResourceAttr(
							ref, "rule.0.ports", "5672,5671,1883,8883,61613,61614"),
						resource.TestCheckResourceAttr(
							ref, "rule.0.log", "true"),
						resource.TestCheckResourceAttr(
							ref, "rule.1.protocol", "udp"),
						resource.TestCheckResourceAttr(
							ref, "rule.1.destination", "192.168.1.101"),
						resource.TestCheckResourceAttr(
							ref, "rule.1.ports", "5674,5673"),
					),
				},

				resource.TestStep{
					Config: securityGroupUpdate,
					Check: resource.ComposeTestCheckFunc(
						testAccCheckASGExists(ref),
						resource.TestCheckResourceAttr(
							ref, "name", asgname),
						resource.TestCheckResourceAttr(
							ref, "rule.#", "3"),
						resource.TestCheckResourceAttr(
							ref, "rule.0.protocol", "tcp"),
						resource.TestCheckResourceAttr(
							ref, "rule.0.destination", "192.168.1.100"),
						resource.TestCheckResourceAttr(
							ref, "rule.0.ports", "61613,61614"),
						resource.TestCheckResourceAttr(
							ref, "rule.1.protocol", "tcp"),
						resource.TestCheckResourceAttr(
							ref, "rule.1.destination", "192.168.1.0/24"),
						resource.TestCheckResourceAttr(
							ref, "rule.1.ports", "61613,61614"),
						resource.TestCheckResourceAttr(
							ref, "rule.2.protocol", "all"),
						resource.TestCheckResourceAttr(
							ref, "rule.2.destination", "0.0.0.0/0"),
						resource.TestCheckResourceAttr(
							ref, "rule.2.ports", ""),
						resource.TestCheckResourceAttr(
							ref, "rule.2.log", "true"),
					),
				},
			},
		})
}

func testAccCheckASGExists(resource string) resource.TestCheckFunc {

	return func(s *terraform.State) error {

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

		asg, err := session.ASGManager().GetASG(id)
		if err != nil {
			return err
		}

		if err := assertEquals(attributes, "name", asg.Name); err != nil {
			return err
		}

		if err := assertListEquals(attributes, "rule", len(asg.Rules),
			func(values map[string]string, i int) (match bool) {

				return values["protocol"] == asg.Rules[i].Protocol &&
					values["destination"] == asg.Rules[i].Destination &&
					values["ports"] == asg.Rules[i].Ports &&
					values["log"] == strconv.FormatBool(asg.Rules[i].Log)

			}); err != nil {
			return err
		}

		return nil
	}
}

func testAccCheckASGDestroy(asgname string) resource.TestCheckFunc {

	return func(s *terraform.State) error {

		session := testAccProvider.Meta().(*cfapi.Session)
		if _, err := session.ASGManager().Read(asgname); err != nil {
			switch err.(type) {
			case *errors.ModelNotFoundError:
				return nil
			default:
				return err
			}
		}
		return fmt.Errorf("asg with name '%s' still exists in cloud foundry", asgname)
	}
}
