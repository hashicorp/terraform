package fastly

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	gofastly "github.com/sethvargo/go-fastly"
)

func TestResourceFastlyFlattenSumologic(t *testing.T) {
	cases := []struct {
		remote []*gofastly.Sumologic
		local  []map[string]interface{}
	}{
		{
			remote: []*gofastly.Sumologic{
				&gofastly.Sumologic{
					Name:              "sumo collector",
					URL:               "https://sumologic.com/collector/1",
					Format:            "log format",
					FormatVersion:     2,
					MessageType:       "classic",
					ResponseCondition: "condition 1",
				},
			},
			local: []map[string]interface{}{
				map[string]interface{}{
					"name":               "sumo collector",
					"url":                "https://sumologic.com/collector/1",
					"format":             "log format",
					"format_version":     2,
					"message_type":       "classic",
					"response_condition": "condition 1",
				},
			},
		},
	}

	for _, c := range cases {
		out := flattenSumologics(c.remote)
		if !reflect.DeepEqual(out, c.local) {
			t.Fatalf("Error matching:\nexpected: %#v\ngot: %#v", c.local, out)
		}
	}
}

func TestAccFastlyServiceV1_sumologic(t *testing.T) {
	var service gofastly.ServiceDetail
	name := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	sumologicName := fmt.Sprintf("sumologic %s", acctest.RandString(3))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckServiceV1Destroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccServiceV1Config_sumologic(name, sumologicName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceV1Exists("fastly_service_v1.foo", &service),
					testAccCheckFastlyServiceV1Attributes_sumologic(&service, name, sumologicName),
				),
			},
		},
	})
}

func testAccCheckFastlyServiceV1Attributes_sumologic(service *gofastly.ServiceDetail, name, sumologic string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if service.Name != name {
			return fmt.Errorf("Bad name, expected (%s), got (%s)", name, service.Name)
		}

		conn := testAccProvider.Meta().(*FastlyClient).conn
		sumologicList, err := conn.ListSumologics(&gofastly.ListSumologicsInput{
			Service: service.ID,
			Version: service.ActiveVersion.Number,
		})

		if err != nil {
			return fmt.Errorf("[ERR] Error looking up Sumologics for (%s), version (%v): %s", service.Name, service.ActiveVersion.Number, err)
		}

		if len(sumologicList) != 1 {
			return fmt.Errorf("Sumologic missing, expected: 1, got: %d", len(sumologicList))
		}

		if sumologicList[0].Name != sumologic {
			return fmt.Errorf("Sumologic name mismatch, expected: %s, got: %#v", sumologic, sumologicList[0].Name)
		}

		return nil
	}
}

func testAccServiceV1Config_sumologic(name, sumologic string) string {
	backendName := fmt.Sprintf("%s.aws.amazon.com", acctest.RandString(3))

	return fmt.Sprintf(`
resource "fastly_service_v1" "foo" {
  name = "%s"

  domain {
    name    = "test.notadomain.com"
    comment = "tf-testing-domain"
  }

  backend {
    address = "%s"
    name    = "tf -test backend"
  }

  sumologic {
  	name = "%s"
  	url = "https://sumologic.com/collector/1"
  	format_version = 2
  }

  force_destroy = true
}`, name, backendName, sumologic)
}
