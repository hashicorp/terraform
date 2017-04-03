package contentful

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	contentful "github.com/tolgaakyuz/contentful-go"
)

func TestAccContentfulContentType_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckContentfulContentTypeDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccContentfulContentTypeConfig,
				Check: resource.TestCheckResourceAttr(
					"contentful_contenttype.mycontenttype", "name", "TF Acc Test CT 1"),
			},
			resource.TestStep{
				Config: testAccContentfulContentTypeUpdateConfig,
				Check: resource.TestCheckResourceAttr(
					"contentful_contenttype.mycontenttype", "name", "TF Acc Test CT name change"),
			},
			resource.TestStep{
				Config: testAccContentfulContentTypeLinkConfig,
				Check: resource.TestCheckResourceAttr(
					"contentful_contenttype.mylinked_contenttype", "name", "TF Acc Test Linked CT"),
			},
		},
	})
}

func testAccCheckContentfulContentTypeExists(n string, contentType *contentful.ContentType) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No content type ID is set")
		}

		spaceID := rs.Primary.Attributes["space_id"]
		if spaceID == "" {
			return fmt.Errorf("No space_id is set")
		}

		client := testAccProvider.Meta().(*contentful.Contentful)

		ct, err := client.ContentTypes.Get(spaceID, rs.Primary.ID)
		if err != nil {
			return err
		}

		*contentType = *ct

		return nil
	}
}

func testAccCheckContentfulContentTypeDestroy(s *terraform.State) (err error) {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "contentful_contenttype" {
			continue
		}

		spaceID := rs.Primary.Attributes["space_id"]
		if spaceID == "" {
			return fmt.Errorf("No space_id is set")
		}

		client := testAccProvider.Meta().(*contentful.Contentful)

		_, err := client.ContentTypes.Get(spaceID, rs.Primary.ID)
		if _, ok := err.(contentful.NotFoundError); ok {
			return nil
		}

		return fmt.Errorf("Content Type still exists with id: %s", rs.Primary.ID)
	}

	return nil
}

var testAccContentfulContentTypeConfig = `
resource "contentful_space" "myspace" {
  name = "TF Acc Test Space"
}

resource "contentful_contenttype" "mycontenttype" {
  space_id = "${contentful_space.myspace.id}"

  name = "TF Acc Test CT 1"
  description = "Terraform Acc Test Content Type"
  display_field = "field1"

  field {
    id = "field1"
    name = "Field 1"
    type = "Text"
    required = true
  }

  field {
    id = "field2"
    name = "Field 2"
    type = "Integer"
    required = false
  }
}
`

var testAccContentfulContentTypeUpdateConfig = `
resource "contentful_contenttype" "mycontenttype" {
  space_id = "${contentful_space.myspace.id}"

  name = "TF Acc Test CT name change"
  description = "Terraform Acc Test Content Type description change"
  display_field = "field1"

  field {
    id = "field1"
    name = "Field 1 name change"
    type = "Text"
    required = true
  }

  field {
    id = "field3"
    name = "Field 3 new field"
    type = "Integer"
    required = true
  }	
}
`
var testAccContentfulContentTypeLinkConfig = `
resource "contentful_contenttype" "mycontenttype" {
  space_id = "${contentful_space.myspace.id}"

  name = "TF Acc Test CT name change"
  description = "Terraform Acc Test Content Type description change"
  display_field = "field1"

  field {
    id = "field1"
    name = "Field 1 name change"
    type = "Text"
    required = true
  }

  field {
    id = "field3"
    name = "Field 3 new field"
    type = "Integer"
    required = true
  }	
}

resource "contentful_contenttype" "mylinked_contenttype" {
  space_id = "${contentful_space.myspace.id}"

  name = "TF Acc Test Linked CT"
  description = "Terraform Acc Test Content Type with links"
  display_field = "asset_field"

  field {
    id = "asset_field"
    name = "Asset Field"
    type = "Array"
		items {
			type = "Link"
			link_type = "Asset"
		}
    required = true
  }

  field {
    id = "entry_link_field"
    name = "Entry Link Field"
    type = "Link"
		link_type = "Entry"
		validations = ["{\"linkContentType\": [\"${contentful_contenttype.mycontenttype.id}\"]}"]
    required = false
  }

}

`
