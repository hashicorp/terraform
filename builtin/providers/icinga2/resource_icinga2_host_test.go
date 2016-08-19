package icinga2

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccCreateBasicHost(t *testing.T) {

	var testAccCreateBasicHost = fmt.Sprintf(`
		resource "icinga2_host" "basic" {
		hostname      = "terraform-test"
		address       = "10.10.40.1"
		check_command = "hostalive"
	}`)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCreateBasicHost,
				Check: resource.ComposeTestCheckFunc(
					VerifyResourceExists(t, "icinga2_host.basic"),
					testAccCheckResourceState("icinga2_host.basic", "hostname", "terraform-test"),
					testAccCheckResourceState("icinga2_host.basic", "address", "10.10.40.1"),
					testAccCheckResourceState("icinga2_host.basic", "check_command", "hostalive"),
				),
			},
		},
	})
}

func TestAccModifyHostname(t *testing.T) {

	var testAccModifyHostname = fmt.Sprintf(`
		resource "icinga2_host" "basic" {
		hostname      = "terraform-test-rename"
		address       = "10.10.40.1"
		check_command = "hostalive"
	  }`)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccModifyHostname,
				Check: resource.ComposeTestCheckFunc(
					VerifyResourceExists(t, "icinga2_host.basic"),
					testAccCheckResourceState("icinga2_host.basic", "hostname", "terraform-test-rename"),
					testAccCheckResourceState("icinga2_host.basic", "address", "10.10.40.1"),
					testAccCheckResourceState("icinga2_host.basic", "check_command", "hostalive"),
				),
			},
		},
	})
}

func TestAccModifyAddress(t *testing.T) {

	var testAccModifyAddress = fmt.Sprintf(`
		resource "icinga2_host" "basic" {
		hostname      = "terraform-test-rename"
		address       = "20.10.40.1"
		check_command = "hostalive"
	  }`)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccModifyAddress,
				Check: resource.ComposeTestCheckFunc(
					VerifyResourceExists(t, "icinga2_host.basic"),
					testAccCheckResourceState("icinga2_host.basic", "hostname", "terraform-test-rename"),
					testAccCheckResourceState("icinga2_host.basic", "address", "20.10.40.1"),
					testAccCheckResourceState("icinga2_host.basic", "check_command", "hostalive"),
				),
			},
		},
	})
}

// Test setting template for Host. NOTE : This assumes the template already exists
func TestAccCreateTemplatedHost(t *testing.T) {

	var testAccCreateTemplatedHost = fmt.Sprintf(`
		resource "icinga2_host" "templated" {
		hostname      = "terraform-test-templated"
		address       = "20.10.40.1"
		check_command = "hostalive"
		templates = [ "bp-host-web" ]
	}`)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCreateTemplatedHost,
				Check: resource.ComposeTestCheckFunc(
					VerifyResourceExists(t, "icinga2_host.templated"),
					testAccCheckResourceState("icinga2_host.templated", "hostname", "terraform-test-templated"),
					testAccCheckResourceState("icinga2_host.templated", "address", "20.10.40.1"),
					testAccCheckResourceState("icinga2_host.templated", "check_command", "hostalive"),
					testAccCheckResourceState("icinga2_host.templated", "templates.#", "1"),
					testAccCheckResourceState("icinga2_host.templated", "templates.0", "bp-host-web"),
				),
			},
		},
	})
}

func TestAccCreateVariableHost(t *testing.T) {

	var testAccCreateVariableHost = fmt.Sprintf(`
		resource "icinga2_host" "variable" {
		hostname = "terraform-test-variable"
		address = "30.10.40.1"
		check_command = "hostalive"
		vars {
		  os = "linux"
		  osver = "1"
		  allowance = "none" }
		}`)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCreateVariableHost,
				Check: resource.ComposeTestCheckFunc(
					VerifyResourceExists(t, "icinga2_host.variable"),
					testAccCheckResourceState("icinga2_host.variable", "hostname", "terraform-test-variable"),
					testAccCheckResourceState("icinga2_host.variable", "address", "30.10.40.1"),
					testAccCheckResourceState("icinga2_host.variable", "check_command", "hostalive"),
					testAccCheckResourceState("icinga2_host.variable", "vars.#", "3"),
					testAccCheckResourceState("icinga2_host.variable", "vars.allowance", "none"),
					testAccCheckResourceState("icinga2_host.variable", "vars.os", "linux"),
					testAccCheckResourceState("icinga2_host.variable", "vars.osver", "1"),
				),
			},
		},
	})
}
