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
	pid := "terrafom-test-" + acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeProjectMetadataDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeProject_basic0_metadata(pid, pname, org, billingId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeProjectExists(
						"google_compute_project_metadata.fizzbuzz", pid, &project),
					testAccCheckComputeProjectMetadataContains(pid, "banana", "orange"),
					testAccCheckComputeProjectMetadataContains(pid, "sofa", "darwinism"),
					testAccCheckComputeProjectMetadataSize(pid, 2),
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
	pid := "terrafom-test-" + acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeProjectMetadataDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeProject_modify0_metadata(pid, pname, org, billingId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeProjectExists(
						"google_compute_project_metadata.fizzbuzz", pid, &project),
					testAccCheckComputeProjectMetadataContains(pid, "paper", "pen"),
					testAccCheckComputeProjectMetadataContains(pid, "genghis_khan", "french bread"),
					testAccCheckComputeProjectMetadataContains(pid, "happy", "smiling"),
					testAccCheckComputeProjectMetadataSize(pid, 3),
				),
			},

			resource.TestStep{
				Config: testAccComputeProject_modify1_metadata(pid, pname, org, billingId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeProjectExists(
						"google_compute_project_metadata.fizzbuzz", pid, &project),
					testAccCheckComputeProjectMetadataContains(pid, "paper", "pen"),
					testAccCheckComputeProjectMetadataContains(pid, "paris", "french bread"),
					testAccCheckComputeProjectMetadataContains(pid, "happy", "laughing"),
					testAccCheckComputeProjectMetadataSize(pid, 3),
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
	pid := "terraform-test-" + acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeProjectMetadataDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeProject_basic0_metadata(pid, pname, org, billingId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeProjectExists(
						"google_compute_project_metadata.fizzbuzz", pid, &project),
					testAccCheckComputeProjectMetadataContains(pid, "banana", "orange"),
					testAccCheckComputeProjectMetadataContains(pid, "sofa", "darwinism"),
					testAccCheckComputeProjectMetadataSize(pid, 2),
				),
			},

			resource.TestStep{
				Config: testAccComputeProject_basic1_metadata(pid, pname, org, billingId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeProjectExists(
						"google_compute_project_metadata.fizzbuzz", pid, &project),
					testAccCheckComputeProjectMetadataContains(pid, "kiwi", "papaya"),
					testAccCheckComputeProjectMetadataContains(pid, "finches", "darwinism"),
					testAccCheckComputeProjectMetadataSize(pid, 2),
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

func testAccCheckComputeProjectExists(n, pid string, project *compute.Project) resource.TestCheckFunc {
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
			pid).Do()
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

func testAccCheckComputeProjectMetadataContains(pid, key, value string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		config := testAccProvider.Meta().(*Config)
		project, err := config.clientCompute.Projects.Get(pid).Do()
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

func testAccCheckComputeProjectMetadataSize(pid string, size int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		config := testAccProvider.Meta().(*Config)
		project, err := config.clientCompute.Projects.Get(pid).Do()
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

func testAccComputeProject_basic0_metadata(pid, name, org, billing string) string {
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
}`, pid, name, org, billing)
}

func testAccComputeProject_basic1_metadata(pid, name, org, billing string) string {
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
}`, pid, name, org, billing)
}

func testAccComputeProject_modify0_metadata(pid, name, org, billing string) string {
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
}`, pid, name, org, billing)
}

func testAccComputeProject_modify1_metadata(pid, name, org, billing string) string {
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
}`, pid, name, org, billing)
}
