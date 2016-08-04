package atlas

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDataSourceArtifact_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDataArtifact_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDataArtifactState("name", "hashicorp/tf-provider-test"),
				),
			},
		},
	})
}

func TestAccDataSourceArtifact_metadata(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDataArtifact_metadata,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDataArtifactState("name", "hashicorp/tf-provider-test"),
					testAccCheckDataArtifactState("id", "x86"),
					testAccCheckDataArtifactState("metadata_full.arch", "x86"),
				),
			},
		},
	})
}

func TestAccDataSourceArtifact_metadataSet(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDataArtifact_metadataSet,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDataArtifactState("name", "hashicorp/tf-provider-test"),
					testAccCheckDataArtifactState("id", "x64"),
					testAccCheckDataArtifactState("metadata_full.arch", "x64"),
				),
			},
		},
	})
}

func TestAccDataSourceArtifact_buildLatest(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDataArtifact_buildLatest,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDataArtifactState("name", "hashicorp/tf-provider-test"),
				),
			},
		},
	})
}

func TestAccDataSourceArtifact_versionAny(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDataArtifact_versionAny,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDataArtifactState("name", "hashicorp/tf-provider-test"),
				),
			},
		},
	})
}

func testAccCheckDataArtifactState(key, value string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["data.atlas_artifact.foobar"]
		if !ok {
			return fmt.Errorf("Not found: %s", "data.atlas_artifact.foobar")
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		p := rs.Primary
		if p.Attributes[key] != value {
			return fmt.Errorf(
				"%s != %s (actual: %s)", key, value, p.Attributes[key])
		}

		return nil
	}
}

const testAccDataArtifact_basic = `
data "atlas_artifact" "foobar" {
	name = "hashicorp/tf-provider-test"
	type = "foo"
}`

const testAccDataArtifact_metadata = `
data "atlas_artifact" "foobar" {
	name = "hashicorp/tf-provider-test"
	type = "foo"
	metadata {
		arch = "x86"
	}
	version = "any"
}`

const testAccDataArtifact_metadataSet = `
data "atlas_artifact" "foobar" {
	name = "hashicorp/tf-provider-test"
	type = "foo"
	metadata_keys = ["arch"]
	version = "any"
}`

const testAccDataArtifact_buildLatest = `
data "atlas_artifact" "foobar" {
	name = "hashicorp/tf-provider-test"
	type = "foo"
	build = "latest"
	metadata {
		arch = "x86"
	}
}`

const testAccDataArtifact_versionAny = `
data "atlas_artifact" "foobar" {
	name = "hashicorp/tf-provider-test"
	type = "foo"
	version = "any"
}`
