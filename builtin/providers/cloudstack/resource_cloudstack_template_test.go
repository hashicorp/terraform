package cloudstack

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/xanzy/go-cloudstack/cloudstack"
)

func TestAccCloudStackTemplate_full(t *testing.T) {
	var template cloudstack.Template

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudStackTemplateDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCloudStackTemplate_options,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStackTemplateExists("cloudstack_template.foo", &template),
					testAccCheckCloudStackTemplateBasicAttributes(&template),
					testAccCheckCloudStackTemplateOptionalAttributes(&template),
				),
			},
		},
	})
}

func testAccCheckCloudStackTemplateExists(n string, template *cloudstack.Template) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No template ID is set")
		}

		cs := testAccProvider.Meta().(*cloudstack.CloudStackClient)
		tmpl, _, err := cs.Template.GetTemplateByID(rs.Primary.ID, "all")

		if err != nil {
			return err
		}

		if tmpl.Id != rs.Primary.ID {
			return fmt.Errorf("Template not found")
		}

		*template = *tmpl

		return nil
	}
}

func testAccCheckCloudStackTemplateDestroy(s *terraform.State) error {
	cs := testAccProvider.Meta().(*cloudstack.CloudStackClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "cloudstack_template" {
			continue
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No template ID is set")
		}

		p := cs.Template.NewDeleteTemplateParams(rs.Primary.ID)
		_, err := cs.Template.DeleteTemplate(p)

		if err != nil {
			return fmt.Errorf(
				"Error deleting template (%s): %s",
				rs.Primary.ID, err)
		}
	}

	return nil
}

var testAccCloudStackTemplate_basic = fmt.Sprintf(`
resource "cloudstack_template" "foo" {
  name = "terraform-acc-test"
  url = "%s"
  hypervisor = "%s"
  os_type = "%s"
  format = "%s"
  zone = "%s"
}
`,
	CLOUDSTACK_TEMPLATE_URL,
	CLOUDSTACK_HYPERVISOR,
	CLOUDSTACK_TEMPLATE_OS_TYPE,
	CLOUDSTACK_TEMPLATE_FORMAT,
	CLOUDSTACK_ZONE)

func testAccCheckCloudStackTemplateBasicAttributes(template *cloudstack.Template) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if template.Name != "terraform-acc-test" {
			return fmt.Errorf("Bad name: %s", template.Name)
		}

		//todo: could add size to schema and check that, would assure we downloaded/initialize the image properly

		if template.Hypervisor != CLOUDSTACK_HYPERVISOR {
			return fmt.Errorf("Bad hypervisor: %s", template.Hypervisor)
		}

		if template.Ostypename != CLOUDSTACK_TEMPLATE_OS_TYPE {
			return fmt.Errorf("Bad os type: %s", template.Ostypename)
		}

		if template.Format != CLOUDSTACK_TEMPLATE_FORMAT {
			return fmt.Errorf("Bad format: %s", template.Format)
		}

		if template.Zonename != CLOUDSTACK_ZONE {
			return fmt.Errorf("Bad zone: %s", template.Zonename)
		}

		return nil
	}
}

//may prove difficult to test isrouting, isfeatured, ispublic, bits so not set here
var testAccCloudStackTemplate_options = fmt.Sprintf(`
resource "cloudstack_template" "foo" {
  name = "terraform-acc-test"
  url = "%s"
  hypervisor = "%s"
  os_type = "%s"
  format = "%s"
  zone = "%s"
  password_enabled = true
  template_tag = "acctest"
  ssh_key_enabled = true
  is_extractable = true
  is_dynamically_scalable = true
  checksum = "%s" 
}
`,
	CLOUDSTACK_TEMPLATE_URL,
	CLOUDSTACK_HYPERVISOR,
	CLOUDSTACK_TEMPLATE_OS_TYPE,
	CLOUDSTACK_TEMPLATE_FORMAT,
	CLOUDSTACK_ZONE,
	CLOUDSTACK_TEMPLATE_CHECKSUM)

func testAccCheckCloudStackTemplateOptionalAttributes(template *cloudstack.Template) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if !template.Passwordenabled {
			return fmt.Errorf("Bad password_enabled: %s", template.Passwordenabled)
		}

		if template.Templatetag != "acctest" {
			return fmt.Errorf("Bad template_tag: %s", template.Templatetag)
		}

		if !template.Sshkeyenabled {
			return fmt.Errorf("Bad ssh_key_enabled: %s", template.Sshkeyenabled)
		}

		if !template.Isextractable {
			return fmt.Errorf("Bad is_extractable: %s", template.Isextractable)
		}

		if !template.Isdynamicallyscalable {
			return fmt.Errorf("Bad is_dynamically_scalable: %s", template.Isdynamicallyscalable)
		}

		if template.Checksum != CLOUDSTACK_TEMPLATE_CHECKSUM {
			return fmt.Errorf("Bad checksum: %s", template.Checksum)
		}

		return nil
	}
}
