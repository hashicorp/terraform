package cloudfoundry

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"code.cloudfoundry.org/cli/cf/errors"

	"github.com/hashicorp/terraform/builtin/providers/cf/cfapi"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

const domainResourceShared = `

data "cf_domain" "apps" {
    sub_domain = "local"
}

resource "cf_domain" "shared" {
    sub_domain = "dev"
	domain = "${data.cf_domain.apps.domain}"
}
`

const domainResourcePrivate = `

resource "cf_domain" "private" {
    name = "pcfdev-org.io"
	org = "%s"
}
`

func TestAccSharedDomain_normal(t *testing.T) {

	_, filename, _, _ := runtime.Caller(0)
	ut := os.Getenv("UNIT_TEST")
	if !testAccEnvironmentSet() || (len(ut) > 0 && ut != filepath.Base(filename)) {
		fmt.Printf("Skipping tests in '%s'.\n", filepath.Base(filename))
		return
	}

	ref := "cf_domain.shared"
	domainname := "dev.pcfdev.io"

	resource.Test(t,
		resource.TestCase{
			PreCheck:     func() { testAccPreCheck(t) },
			Providers:    testAccProviders,
			CheckDestroy: testAccCheckSharedDomainDestroy(domainname),
			Steps: []resource.TestStep{

				resource.TestStep{
					Config: domainResourceShared,
					Check: resource.ComposeTestCheckFunc(
						checkShareDomainExists(ref),
						resource.TestCheckResourceAttr(
							ref, "name", "dev.pcfdev.io"),
						resource.TestCheckResourceAttr(
							ref, "sub_domain", "dev"),
						resource.TestCheckResourceAttr(
							ref, "domain", "pcfdev.io"),
					),
				},
			},
		})
}

func TestAccPrivateDomain_normal(t *testing.T) {

	_, filename, _, _ := runtime.Caller(0)
	ut := os.Getenv("UNIT_TEST")
	if !testAccEnvironmentSet() || (len(ut) > 0 && ut != filepath.Base(filename)) {
		fmt.Printf("Skipping tests in '%s'.\n", filepath.Base(filename))
		return
	}

	ref := "cf_domain.private"
	domainname := "pcfdev-org.io"
	orgID := defaultPcfDevOrgID()

	resource.Test(t,
		resource.TestCase{
			PreCheck:     func() { testAccPreCheck(t) },
			Providers:    testAccProviders,
			CheckDestroy: testAccCheckPrivateDomainDestroy(domainname),
			Steps: []resource.TestStep{

				resource.TestStep{
					Config: fmt.Sprintf(domainResourcePrivate, orgID),
					Check: resource.ComposeTestCheckFunc(
						checkPrivateDomainExists(ref),
						resource.TestCheckResourceAttr(
							ref, "name", "pcfdev-org.io"),
						resource.TestCheckResourceAttr(
							ref, "sub_domain", "pcfdev-org"),
						resource.TestCheckResourceAttr(
							ref, "domain", "io"),
						resource.TestCheckResourceAttr(
							ref, "org", orgID),
					),
				},
			},
		})
}

func checkShareDomainExists(resource string) resource.TestCheckFunc {

	return func(s *terraform.State) error {

		session := testAccProvider.Meta().(*cfapi.Session)

		rs, ok := s.RootModule().Resources[resource]
		if !ok {
			return fmt.Errorf("domain '%s' not found in terraform state", resource)
		}

		session.Log.DebugMessage(
			"terraform state for resource '%s': %# v",
			resource, rs)

		id := rs.Primary.ID
		attributes := rs.Primary.Attributes
		name := attributes["name"]

		dm := session.DomainManager()
		domainFields, err := dm.FindSharedByName(name)
		if err != nil {
			return err
		}

		if id != domainFields.GUID {
			return fmt.Errorf("expecting domain guid to be '%s' but got '%session'", id, domainFields.GUID)
		}
		return nil
	}
}

func checkPrivateDomainExists(resource string) resource.TestCheckFunc {

	return func(s *terraform.State) error {

		session := testAccProvider.Meta().(*cfapi.Session)

		rs, ok := s.RootModule().Resources[resource]
		if !ok {
			return fmt.Errorf("domain '%s' not found in terraform state", resource)
		}

		session.Log.DebugMessage(
			"terraform state for resource '%s': %# v",
			resource, rs)

		id := rs.Primary.ID
		attributes := rs.Primary.Attributes
		name := attributes["name"]

		dm := session.DomainManager()
		domainFields, err := dm.FindPrivateByName(name)
		if err != nil {
			return err
		}

		if id != domainFields.GUID {
			return fmt.Errorf("expecting domain guid to be '%s' but got '%session'", id, domainFields.GUID)
		}
		if err := assertEquals(attributes, "org", domainFields.OwningOrganizationGUID); err != nil {
			return err
		}
		return nil
	}
}

func testAccCheckSharedDomainDestroy(domainname string) resource.TestCheckFunc {

	return func(s *terraform.State) error {

		session := testAccProvider.Meta().(*cfapi.Session)
		_, err := session.DomainManager().FindSharedByName(domainname)
		switch err.(type) {
		case *errors.ModelNotFoundError:
			return nil
		}
		return fmt.Errorf("shared domain with name '%s' still exists in cloud foundry", domainname)
	}
}

func testAccCheckPrivateDomainDestroy(domainname string) resource.TestCheckFunc {

	return func(s *terraform.State) error {
		session := testAccProvider.Meta().(*cfapi.Session)
		if _, err := session.DomainManager().FindPrivateByName(domainname); err != nil {
			switch err.(type) {
			case *errors.ModelNotFoundError:
				return nil
			default:
				return err
			}
		}
		return fmt.Errorf("domain with name '%s' still exists in cloud foundry", domainname)
	}
}
