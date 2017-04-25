package atlas

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccArtifact_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccArtifact_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckArtifactState("name", "hashicorp/tf-provider-test"),
				),
			},
		},
	})
}

func TestAccArtifact_metadata(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccArtifact_metadata,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckArtifactState("name", "hashicorp/tf-provider-test"),
					testAccCheckArtifactState("id", "x86"),
					testAccCheckArtifactState("metadata_full.arch", "x86"),
				),
			},
		},
	})
}

func TestAccArtifact_metadataSet(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccArtifact_metadataSet,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckArtifactState("name", "hashicorp/tf-provider-test"),
					testAccCheckArtifactState("id", "x64"),
					testAccCheckArtifactState("metadata_full.arch", "x64"),
				),
			},
		},
	})
}

func TestAccArtifact_buildLatest(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccArtifact_buildLatest,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckArtifactState("name", "hashicorp/tf-provider-test"),
				),
			},
		},
	})
}

func TestAccArtifact_versionAny(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccArtifact_versionAny,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckArtifactState("name", "hashicorp/tf-provider-test"),
				),
			},
		},
	})
}

func testAccCheckArtifactState(key, value string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["atlas_artifact.foobar"]
		if !ok {
			return fmt.Errorf("Not found: %s", "atlas_artifact.foobar")
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

const testAccArtifact_basic = `
resource "atlas_artifact" "foobar" {
	name = "hashicorp/tf-provider-test"
	type = "foo"
}`

const testAccArtifact_metadata = `
resource "atlas_artifact" "foobar" {
	name = "hashicorp/tf-provider-test"
	type = "foo"
	metadata {
		arch = "x86"
	}
	version = "any"
}`

const testAccArtifact_metadataSet = `
resource "atlas_artifact" "foobar" {
	name = "hashicorp/tf-provider-test"
	type = "foo"
	metadata_keys = ["arch"]
	version = "any"
}`

const testAccArtifact_buildLatest = `
resource "atlas_artifact" "foobar" {
	name = "hashicorp/tf-provider-test"
	type = "foo"
	build = "latest"
	metadata {
		arch = "x86"
	}
}`

const testAccArtifact_versionAny = `
resource "atlas_artifact" "foobar" {
	name = "hashicorp/tf-provider-test"
	type = "foo"
	version = "any"
}`
