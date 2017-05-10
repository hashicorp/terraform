package google

import (
	"errors"
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccGoogleContainerEngineVersions_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckGoogleContainerEngineVersionsConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGoogleContainerEngineVersionsMeta("data.google_container_engine_versions.versions"),
				),
			},
		},
	})
}

func testAccCheckGoogleContainerEngineVersionsMeta(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Can't find versions data source: %s", n)
		}

		if rs.Primary.ID == "" {
			return errors.New("versions data source ID not set.")
		}

		nodeCount, ok := rs.Primary.Attributes["valid_node_versions.#"]
		if !ok {
			return errors.New("can't find 'valid_node_versions' attribute")
		}

		noOfNodes, err := strconv.Atoi(nodeCount)
		if err != nil {
			return errors.New("failed to read number of valid node versions")
		}
		if noOfNodes < 2 {
			return fmt.Errorf("expected at least 2 valid node versions, received %d, this is most likely a bug",
				noOfNodes)
		}

		for i := 0; i < noOfNodes; i++ {
			idx := "valid_node_versions." + strconv.Itoa(i)
			v, ok := rs.Primary.Attributes[idx]
			if !ok {
				return fmt.Errorf("valid node versions list is corrupt (%q not found), this is definitely a bug", idx)
			}
			if len(v) < 1 {
				return fmt.Errorf("Empty node version (%q), this is definitely a bug", idx)
			}
		}

		masterCount, ok := rs.Primary.Attributes["valid_master_versions.#"]
		if !ok {
			return errors.New("can't find 'valid_master_versions' attribute")
		}

		noOfMasters, err := strconv.Atoi(masterCount)
		if err != nil {
			return errors.New("failed to read number of valid master versions")
		}
		if noOfMasters < 2 {
			return fmt.Errorf("expected at least 2 valid master versions, received %d, this is most likely a bug",
				noOfMasters)
		}

		for i := 0; i < noOfMasters; i++ {
			idx := "valid_master_versions." + strconv.Itoa(i)
			v, ok := rs.Primary.Attributes[idx]
			if !ok {
				return fmt.Errorf("valid master versions list is corrupt (%q not found), this is definitely a bug", idx)
			}
			if len(v) < 1 {
				return fmt.Errorf("Empty master version (%q), this is definitely a bug", idx)
			}
		}

		return nil
	}
}

var testAccCheckGoogleContainerEngineVersionsConfig = `
data "google_container_engine_versions" "versions" {
  zone = "us-central1-b"
}
`
