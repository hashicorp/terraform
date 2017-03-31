package elasticsearch

import (
	"context"
	"fmt"
	"testing"

	elastic "gopkg.in/olivere/elastic.v5"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccElasticsearchIndexTemplate(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckElasticsearchIndexTemplateDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccElasticsearchIndexTemplate,
				Check: resource.ComposeTestCheckFunc(
					testCheckElasticsearchIndexTemplateExists("elasticsearch_index_template.test"),
				),
			},
		},
	})
}

func testCheckElasticsearchIndexTemplateExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No index template ID is set")
		}

		conn := testAccProvider.Meta().(*elastic.Client)
		_, err := conn.IndexGetTemplate(rs.Primary.ID).Do(context.TODO())
		if err != nil {
			return err
		}

		return nil
	}
}

func testCheckElasticsearchIndexTemplateDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*elastic.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "elasticsearch_index_template" {
			continue
		}

		_, err := conn.IndexGetTemplate(rs.Primary.ID).Do(context.TODO())
		if err != nil {
			return nil
		}

		return fmt.Errorf("Index template %q still exists", rs.Primary.ID)
	}

	return nil
}

var testAccElasticsearchIndexTemplate = `
resource "elasticsearch_index_template" "test" {
  name = "terraform-test"
  body = <<EOF
{
  "template": "logstash-*",
  "version": 50001,
  "settings": {
    "index.refresh_interval": "5s"
  },
  "mappings": {
    "_default_": {
      "_all": {"enabled": true, "norms": false},
      "dynamic_templates": [ {
        "message_field": {
          "path_match": "message",
          "match_mapping_type": "string",
          "mapping": {
            "type": "text",
            "norms": false
          }
        }
      }, {
        "string_fields": {
          "match": "*",
          "match_mapping_type": "string",
          "mapping": {
            "type": "text", "norms": false,
            "fields": {
              "keyword": { "type": "keyword" }
            }
          }
        }
      } ],
      "properties": {
        "@timestamp": { "type": "date", "include_in_all": false },
        "@version": { "type": "keyword", "include_in_all": false },
        "geoip" : {
          "dynamic": true,
          "properties": {
            "ip": { "type": "ip" },
            "location": { "type": "geo_point" },
            "latitude": { "type": "half_float" },
            "longitude": { "type": "half_float" }
          }
        }
      }
    }
  }
}
EOF
}
`
