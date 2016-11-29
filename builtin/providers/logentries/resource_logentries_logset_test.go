package logentries

import (
	"fmt"
	lexp "github.com/hashicorp/terraform/builtin/providers/logentries/expect"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/logentries/le_goclient"
	"testing"
)

type LogSetResource struct {
	Name     string `tfresource:"name"`
	Location string `tfresource:"location"`
}

func TestAccLogentriesLogSet_Basic(t *testing.T) {
	var logSetResource LogSetResource

	logSetName := fmt.Sprintf("terraform-test-%s", acctest.RandString(8))
	testAccLogentriesLogSetConfig := fmt.Sprintf(`
		resource "logentries_logset" "test_logset" {
			name = "%s"
			location = "terraform.io"
		}
	`, logSetName)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLogentriesLogSetDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccLogentriesLogSetConfig,
				Check: lexp.TestCheckResourceExpectation(
					"logentries_logset.test_logset",
					&logSetResource,
					testAccCheckLogentriesLogSetExists,
					map[string]lexp.TestExpectValue{
						"name":     lexp.Equals(logSetName),
						"location": lexp.Equals("terraform.io"),
					},
				),
			},
		},
	})
}

func TestAccLogentriesLogSet_NoLocation(t *testing.T) {
	var logSetResource LogSetResource

	logSetName := fmt.Sprintf("terraform-test-%s", acctest.RandString(8))
	testAccLogentriesLogSetConfig := fmt.Sprintf(`
		resource "logentries_logset" "test_logset" {
			name = "%s"
		}
	`, logSetName)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLogentriesLogSetDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccLogentriesLogSetConfig,
				Check: lexp.TestCheckResourceExpectation(
					"logentries_logset.test_logset",
					&logSetResource,
					testAccCheckLogentriesLogSetExists,
					map[string]lexp.TestExpectValue{
						"name":     lexp.Equals(logSetName),
						"location": lexp.Equals("nonlocation"),
					},
				),
			},
		},
	})
}

func testAccCheckLogentriesLogSetDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*logentries.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "logentries_logset" {
			continue
		}

		resp, err := client.LogSet.Read(logentries.LogSetReadRequest{Key: rs.Primary.ID})

		if err == nil {
			return fmt.Errorf("Log set still exists: %#v", resp)
		}
	}

	return nil
}

func testAccCheckLogentriesLogSetExists(resource string, fact interface{}) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resource]

		if !ok {
			return fmt.Errorf("Not found: %s", resource)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No LogSet Key is set")
		}

		client := testAccProvider.Meta().(*logentries.Client)

		resp, err := client.LogSet.Read(logentries.LogSetReadRequest{Key: rs.Primary.ID})

		if err != nil {
			return err
		}

		res := fact.(*LogSetResource)
		res.Location = resp.Location
		res.Name = resp.Name

		return nil
	}
}
