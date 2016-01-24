package google

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"google.golang.org/api/compute/v1"
)

// Add two key value pairs
func TestAccComputeProjectMetadata_basic(t *testing.T) {
	var project compute.Project

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeProjectMetadataDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeProject_basic0_metadata,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeProjectExists(
						"google_compute_project_metadata.fizzbuzz", &project),
					testAccCheckComputeProjectMetadataContains(&project, "banana", "orange"),
					testAccCheckComputeProjectMetadataContains(&project, "sofa", "darwinism"),
					testAccCheckComputeProjectMetadataSize(&project, 2),
				),
			},
		},
	})
}

// Add three key value pairs, then replace one and modify a second
func TestAccComputeProjectMetadata_modify_1(t *testing.T) {
	var project compute.Project

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeProjectMetadataDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeProject_modify0_metadata,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeProjectExists(
						"google_compute_project_metadata.fizzbuzz", &project),
					testAccCheckComputeProjectMetadataContains(&project, "paper", "pen"),
					testAccCheckComputeProjectMetadataContains(&project, "genghis_khan", "french bread"),
					testAccCheckComputeProjectMetadataContains(&project, "happy", "smiling"),
					testAccCheckComputeProjectMetadataSize(&project, 3),
				),
			},

			resource.TestStep{
				Config: testAccComputeProject_modify1_metadata,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeProjectExists(
						"google_compute_project_metadata.fizzbuzz", &project),
					testAccCheckComputeProjectMetadataContains(&project, "paper", "pen"),
					testAccCheckComputeProjectMetadataContains(&project, "paris", "french bread"),
					testAccCheckComputeProjectMetadataContains(&project, "happy", "laughing"),
					testAccCheckComputeProjectMetadataSize(&project, 3),
				),
			},
		},
	})
}

// Add two key value pairs, and replace both
func TestAccComputeProjectMetadata_modify_2(t *testing.T) {
	var project compute.Project

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeProjectMetadataDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeProject_basic0_metadata,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeProjectExists(
						"google_compute_project_metadata.fizzbuzz", &project),
					testAccCheckComputeProjectMetadataContains(&project, "banana", "orange"),
					testAccCheckComputeProjectMetadataContains(&project, "sofa", "darwinism"),
					testAccCheckComputeProjectMetadataSize(&project, 2),
				),
			},

			resource.TestStep{
				Config: testAccComputeProject_basic1_metadata,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeProjectExists(
						"google_compute_project_metadata.fizzbuzz", &project),
					testAccCheckComputeProjectMetadataContains(&project, "kiwi", "papaya"),
					testAccCheckComputeProjectMetadataContains(&project, "finches", "darwinism"),
					testAccCheckComputeProjectMetadataSize(&project, 2),
				),
			},
		},
	})
}

func testAccCheckComputeProjectMetadataDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)

	project, err := config.clientCompute.Projects.Get(config.Project).Do()
	if err == nil && len(project.CommonInstanceMetadata.Items) > 0 {
		return fmt.Errorf("Error, metadata items still exist")
	}

	return nil
}

func testAccCheckComputeProjectExists(n string, project *compute.Project) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)

		found, err := config.clientCompute.Projects.Get(
			config.Project).Do()
		if err != nil {
			return err
		}

		if "common_metadata" != rs.Primary.ID {
			return fmt.Errorf("Common metadata not found, found %s", rs.Primary.ID)
		}

		*project = *found

		return nil
	}
}

func testAccCheckComputeProjectMetadataContains(project *compute.Project, key string, value string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		config := testAccProvider.Meta().(*Config)
		project, err := config.clientCompute.Projects.Get(config.Project).Do()
		if err != nil {
			return fmt.Errorf("Error, failed to load project service for %s: %s", config.Project, err)
		}

		for _, kv := range project.CommonInstanceMetadata.Items {
			if kv.Key == key {
				if kv.Value != nil && *kv.Value == value {
					return nil
				} else {
					return fmt.Errorf("Error, key value mismatch, wanted (%s, %s), got (%s, %s)",
						key, value, kv.Key, *kv.Value)
				}
			}
		}

		return fmt.Errorf("Error, key %s not present", key)
	}
}

func testAccCheckComputeProjectMetadataSize(project *compute.Project, size int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		config := testAccProvider.Meta().(*Config)
		project, err := config.clientCompute.Projects.Get(config.Project).Do()
		if err != nil {
			return fmt.Errorf("Error, failed to load project service for %s: %s", config.Project, err)
		}

		if size > len(project.CommonInstanceMetadata.Items) {
			return fmt.Errorf("Error, expected at least %d metadata items, got %d", size,
				len(project.CommonInstanceMetadata.Items))
		}

		return nil
	}
}

const testAccComputeProject_basic0_metadata = `
resource "google_compute_project_metadata" "fizzbuzz" {
	metadata {
		banana = "orange"
		sofa = "darwinism"
	}
}`

const testAccComputeProject_basic1_metadata = `
resource "google_compute_project_metadata" "fizzbuzz" {
	metadata {
		kiwi = "papaya"
		finches = "darwinism"
	}
}`

const testAccComputeProject_modify0_metadata = `
resource "google_compute_project_metadata" "fizzbuzz" {
	metadata {
		paper = "pen"
		genghis_khan = "french bread"
		happy = "smiling"
	}
}`

const testAccComputeProject_modify1_metadata = `
resource "google_compute_project_metadata" "fizzbuzz" {
	metadata {
		paper = "pen"
		paris = "french bread"
		happy = "laughing"
	}
}`
