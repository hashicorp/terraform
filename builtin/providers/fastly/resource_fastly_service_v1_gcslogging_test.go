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

func TestResourceFastlyFlattenGCS(t *testing.T) {
	cases := []struct {
		remote []*gofastly.GCS
		local  []map[string]interface{}
	}{
		{
			remote: []*gofastly.GCS{
				&gofastly.GCS{
					Name:      "GCS collector",
					User:      "email@example.com",
					Bucket:    "bucketName",
					SecretKey: "secretKey",
					Format:    "log format",
					Period:    3600,
					GzipLevel: 0,
				},
			},
			local: []map[string]interface{}{
				map[string]interface{}{
					"name":        "GCS collector",
					"email":       "email@example.com",
					"bucket_name": "bucketName",
					"secret_key":  "secretKey",
					"format":      "log format",
					"period":      3600,
					"gzip_level":  0,
				},
			},
		},
	}

	for _, c := range cases {
		out := flattenGCS(c.remote)
		if !reflect.DeepEqual(out, c.local) {
			t.Fatalf("Error matching:\nexpected: %#v\ngot: %#v", c.local, out)
		}
	}
}

func TestAccFastlyServiceV1_gcslogging(t *testing.T) {
	var service gofastly.ServiceDetail
	name := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	gcsName := fmt.Sprintf("gcs %s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckServiceV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccServiceV1Config_gcs(name, gcsName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceV1Exists("fastly_service_v1.foo", &service),
					testAccCheckFastlyServiceV1Attributes_gcs(&service, name, gcsName),
				),
			},
		},
	})
}

func testAccCheckFastlyServiceV1Attributes_gcs(service *gofastly.ServiceDetail, name, gcsName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if service.Name != name {
			return fmt.Errorf("Bad name, expected (%s), got (%s)", name, service.Name)
		}

		conn := testAccProvider.Meta().(*FastlyClient).conn
		gcsList, err := conn.ListGCSs(&gofastly.ListGCSsInput{
			Service: service.ID,
			Version: service.ActiveVersion.Number,
		})

		if err != nil {
			return fmt.Errorf("[ERR] Error looking up GCSs for (%s), version (%v): %s", service.Name, service.ActiveVersion.Number, err)
		}

		if len(gcsList) != 1 {
			return fmt.Errorf("GCS missing, expected: 1, got: %d", len(gcsList))
		}

		if gcsList[0].Name != gcsName {
			return fmt.Errorf("GCS name mismatch, expected: %s, got: %#v", gcsName, gcsList[0].Name)
		}

		return nil
	}
}

func testAccServiceV1Config_gcs(name, gcsName string) string {
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

	gcslogging {
	  name =  "%s"
		email = "email@example.com",
		bucket_name = "bucketName",
		secret_key = "secretKey",
		format = "log format",
		response_condition = "",
	}

  force_destroy = true
}`, name, backendName, gcsName)
}
