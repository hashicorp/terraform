package google

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"google.golang.org/api/compute/v1"
)

// Add two key value pairs
func TestAccComputeProjectMetadata_basic(t *testing.T) {
	skipIfEnvNotSet(t,
		[]string{
			"GOOGLE_ORG",
			"GOOGLE_BILLING_ACCOUNT",
		}...,
	)

	billingId := os.Getenv("GOOGLE_BILLING_ACCOUNT")
	var project compute.Project
	projectID := "terrafom-test-" + acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeProjectMetadataDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeProject_basic0_metadata(projectID, pname, org, billingId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeProjectExists(
						"google_compute_project_metadata.fizzbuzz", projectID, &project),
					testAccCheckComputeProjectMetadataContains(projectID, "banana", "orange"),
					testAccCheckComputeProjectMetadataContains(projectID, "sofa", "darwinism"),
					testAccCheckComputeProjectMetadataSize(projectID, 2),
				),
			},
		},
	})
}

// Add three key value pairs, then replace one and modify a second
func TestAccComputeProjectMetadata_modify_1(t *testing.T) {
	skipIfEnvNotSet(t,
		[]string{
			"GOOGLE_ORG",
			"GOOGLE_BILLING_ACCOUNT",
		}...,
	)

	billingId := os.Getenv("GOOGLE_BILLING_ACCOUNT")
	var project compute.Project
	projectID := "terrafom-test-" + acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeProjectMetadataDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeProject_modify0_metadata(projectID, pname, org, billingId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeProjectExists(
						"google_compute_project_metadata.fizzbuzz", projectID, &project),
					testAccCheckComputeProjectMetadataContains(projectID, "paper", "pen"),
					testAccCheckComputeProjectMetadataContains(projectID, "genghis_khan", "french bread"),
					testAccCheckComputeProjectMetadataContains(projectID, "happy", "smiling"),
					testAccCheckComputeProjectMetadataSize(projectID, 3),
				),
			},

			resource.TestStep{
				Config: testAccComputeProject_modify1_metadata(projectID, pname, org, billingId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeProjectExists(
						"google_compute_project_metadata.fizzbuzz", projectID, &project),
					testAccCheckComputeProjectMetadataContains(projectID, "paper", "pen"),
					testAccCheckComputeProjectMetadataContains(projectID, "paris", "french bread"),
					testAccCheckComputeProjectMetadataContains(projectID, "happy", "laughing"),
					testAccCheckComputeProjectMetadataSize(projectID, 3),
				),
			},
		},
	})
}

// Add two key value pairs, and replace both
func TestAccComputeProjectMetadata_modify_2(t *testing.T) {
	skipIfEnvNotSet(t,
		[]string{
			"GOOGLE_ORG",
			"GOOGLE_BILLING_ACCOUNT",
		}...,
	)

	billingId := os.Getenv("GOOGLE_BILLING_ACCOUNT")
	var project compute.Project
	projectID := "terraform-test-" + acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeProjectMetadataDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeProject_basic0_metadata(projectID, pname, org, billingId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeProjectExists(
						"google_compute_project_metadata.fizzbuzz", projectID, &project),
					testAccCheckComputeProjectMetadataContains(projectID, "banana", "orange"),
					testAccCheckComputeProjectMetadataContains(projectID, "sofa", "darwinism"),
					testAccCheckComputeProjectMetadataSize(projectID, 2),
				),
			},

			resource.TestStep{
				Config: testAccComputeProject_basic1_metadata(projectID, pname, org, billingId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeProjectExists(
						"google_compute_project_metadata.fizzbuzz", projectID, &project),
					testAccCheckComputeProjectMetadataContains(projectID, "kiwi", "papaya"),
					testAccCheckComputeProjectMetadataContains(projectID, "finches", "darwinism"),
					testAccCheckComputeProjectMetadataSize(projectID, 2),
				),
			},
		},
	})
}

func testAccCheckComputeProjectMetadataDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "google_compute_project_metadata" {
			continue
		}

		project, err := config.clientCompute.Projects.Get(rs.Primary.ID).Do()
		if err == nil && len(project.CommonInstanceMetadata.Items) > 0 {
			return fmt.Errorf("Error, metadata items still exist in %s", rs.Primary.ID)
		}
	}

	return nil
}

func testAccCheckComputeProjectExists(n, projectID string, project *compute.Project) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)

		found, err := config.clientCompute.Projects.Get(projectID).Do()
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

func testAccCheckComputeProjectMetadataContains(projectID, key, value string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		config := testAccProvider.Meta().(*Config)
		project, err := config.clientCompute.Projects.Get(projectID).Do()
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

		return fmt.Errorf("Error, key %s not present in %s", key, project.SelfLink)
	}
}

func testAccCheckComputeProjectMetadataSize(projectID string, size int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		config := testAccProvider.Meta().(*Config)
		project, err := config.clientCompute.Projects.Get(projectID).Do()
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

func testAccComputeProject_basic0_metadata(projectID, name, org, billing string) string {
	return fmt.Sprintf(`
resource "google_project" "project" {
  project_id = "%s"
  name = "%s"
  org_id = "%s"
  billing_account = "%s"
}

resource "google_project_services" "services" {
  project = "${google_project.project.project_id}"
  services = ["compute-component.googleapis.com"]
}

resource "google_compute_project_metadata" "fizzbuzz" {
  project = "${google_project.project.project_id}"
  metadata {
    banana = "orange"
    sofa = "darwinism"
  }
  depends_on = ["google_project_services.services"]
}`, projectID, name, org, billing)
}

func testAccComputeProject_basic1_metadata(projectID, name, org, billing string) string {
	return fmt.Sprintf(`
resource "google_project" "project" {
  project_id = "%s"
  name = "%s"
  org_id = "%s"
  billing_account = "%s"
}

resource "google_project_services" "services" {
  project = "${google_project.project.project_id}"
  services = ["compute-component.googleapis.com"]
}

resource "google_compute_project_metadata" "fizzbuzz" {
  project = "${google_project.project.project_id}"
  metadata {
    kiwi = "papaya"
    finches = "darwinism"
  }
  depends_on = ["google_project_services.services"]
}`, projectID, name, org, billing)
}

func testAccComputeProject_modify0_metadata(projectID, name, org, billing string) string {
	return fmt.Sprintf(`
resource "google_project" "project" {
  project_id = "%s"
  name = "%s"
  org_id = "%s"
  billing_account = "%s"
}

resource "google_project_services" "services" {
  project = "${google_project.project.project_id}"
  services = ["compute-component.googleapis.com"]
}

resource "google_compute_project_metadata" "fizzbuzz" {
  project = "${google_project.project.project_id}"
  metadata {
    paper = "pen"
    genghis_khan = "french bread"
    happy = "smiling"
  }
  depends_on = ["google_project_services.services"]
}`, projectID, name, org, billing)
}

func testAccComputeProject_modify1_metadata(projectID, name, org, billing string) string {
	return fmt.Sprintf(`
resource "google_project" "project" {
  project_id = "%s"
  name = "%s"
  org_id = "%s"
  billing_account = "%s"
}

resource "google_project_services" "services" {
  project = "${google_project.project.project_id}"
  services = ["compute-component.googleapis.com"]
}

resource "google_compute_project_metadata" "fizzbuzz" {
  project = "${google_project.project.project_id}"
  metadata {
    paper = "pen"
    paris = "french bread"
    happy = "laughing"
  }
  depends_on = ["google_project_services.services"]
}`, projectID, name, org, billing)
}
