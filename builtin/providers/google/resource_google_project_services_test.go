package google

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"reflect"
	"sort"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"google.golang.org/api/servicemanagement/v1"
)

// Test that services can be enabled and disabled on a project
func TestAccGoogleProjectServices_basic(t *testing.T) {
	pid := "terraform-" + acctest.RandString(10)
	services1 := []string{"iam.googleapis.com", "cloudresourcemanager.googleapis.com"}
	services2 := []string{"cloudresourcemanager.googleapis.com"}
	oobService := "iam.googleapis.com"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			// Create a new project with some services
			resource.TestStep{
				Config: testAccGoogleProjectAssociateServicesBasic(services1, pid, pname, org),
				Check: resource.ComposeTestCheckFunc(
					testProjectServicesMatch(services1, pid),
				),
			},
			// Update services to remove one
			resource.TestStep{
				Config: testAccGoogleProjectAssociateServicesBasic(services2, pid, pname, org),
				Check: resource.ComposeTestCheckFunc(
					testProjectServicesMatch(services2, pid),
				),
			},
			// Add a service out-of-band and ensure it is removed
			resource.TestStep{
				PreConfig: func() {
					config := testAccProvider.Meta().(*Config)
					enableService(oobService, pid, config)
				},
				Config: testAccGoogleProjectAssociateServicesBasic(services2, pid, pname, org),
				Check: resource.ComposeTestCheckFunc(
					testProjectServicesMatch(services2, pid),
				),
			},
		},
	})
}

// Test that services are authoritative when a project has existing
// sevices not represented in config
func TestAccGoogleProjectServices_authoritative(t *testing.T) {
	pid := "terraform-" + acctest.RandString(10)
	services := []string{"cloudresourcemanager.googleapis.com"}
	oobService := "iam.googleapis.com"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			// Create a new project with no services
			resource.TestStep{
				Config: testAccGoogleProject_create(pid, pname, org),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGoogleProjectExists("google_project.acceptance", pid),
				),
			},
			// Add a service out-of-band, then apply a config that creates a service.
			// It should remove the out-of-band service.
			resource.TestStep{
				PreConfig: func() {
					config := testAccProvider.Meta().(*Config)
					enableService(oobService, pid, config)
				},
				Config: testAccGoogleProjectAssociateServicesBasic(services, pid, pname, org),
				Check: resource.ComposeTestCheckFunc(
					testProjectServicesMatch(services, pid),
				),
			},
		},
	})
}

// Test that services are authoritative when a project has existing
// sevices, some which are represented in the config and others
// that are not
func TestAccGoogleProjectServices_authoritative2(t *testing.T) {
	pid := "terraform-" + acctest.RandString(10)
	oobServices := []string{"iam.googleapis.com", "cloudresourcemanager.googleapis.com"}
	services := []string{"iam.googleapis.com"}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			// Create a new project with no services
			resource.TestStep{
				Config: testAccGoogleProject_create(pid, pname, org),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGoogleProjectExists("google_project.acceptance", pid),
				),
			},
			// Add a service out-of-band, then apply a config that creates a service.
			// It should remove the out-of-band service.
			resource.TestStep{
				PreConfig: func() {
					config := testAccProvider.Meta().(*Config)
					for _, s := range oobServices {
						enableService(s, pid, config)
					}
				},
				Config: testAccGoogleProjectAssociateServicesBasic(services, pid, pname, org),
				Check: resource.ComposeTestCheckFunc(
					testProjectServicesMatch(services, pid),
				),
			},
		},
	})
}

// Test that services that can't be enabled on their own (such as dataproc-control.googleapis.com)
// don't end up causing diffs when they are enabled as a side-effect of a different service's
// enablement.
func TestAccGoogleProjectServices_ignoreUnenablableServices(t *testing.T) {
	skipIfEnvNotSet(t,
		[]string{
			"GOOGLE_ORG",
			"GOOGLE_BILLING_ACCOUNT",
		}...,
	)

	billingId := os.Getenv("GOOGLE_BILLING_ACCOUNT")
	pid := "terraform-" + acctest.RandString(10)
	services := []string{
		"dataproc.googleapis.com",
		// The following services are enabled as a side-effect of dataproc's enablement
		"storage-component.googleapis.com",
		"deploymentmanager.googleapis.com",
		"replicapool.googleapis.com",
		"replicapoolupdater.googleapis.com",
		"resourceviews.googleapis.com",
		"compute-component.googleapis.com",
		"container.googleapis.com",
		"containerregistry.googleapis.com",
		"storage-api.googleapis.com",
		"pubsub.googleapis.com",
	}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccGoogleProjectAssociateServicesBasic_withBilling(services, pid, pname, org, billingId),
				Check: resource.ComposeTestCheckFunc(
					testProjectServicesMatch(services, pid),
				),
			},
		},
	})
}

func TestAccGoogleProjectServices_manyServices(t *testing.T) {
	skipIfEnvNotSet(t,
		[]string{
			"GOOGLE_ORG",
			"GOOGLE_BILLING_ACCOUNT",
		}...,
	)

	billingId := os.Getenv("GOOGLE_BILLING_ACCOUNT")
	pid := "terraform-" + acctest.RandString(10)
	services := []string{
		"bigquery-json.googleapis.com",
		"cloudbuild.googleapis.com",
		"cloudfunctions.googleapis.com",
		"cloudresourcemanager.googleapis.com",
		"cloudtrace.googleapis.com",
		"compute-component.googleapis.com",
		"container.googleapis.com",
		"containerregistry.googleapis.com",
		"dataflow.googleapis.com",
		"dataproc.googleapis.com",
		"deploymentmanager.googleapis.com",
		"dns.googleapis.com",
		"endpoints.googleapis.com",
		"iam.googleapis.com",
		"logging.googleapis.com",
		"ml.googleapis.com",
		"monitoring.googleapis.com",
		"pubsub.googleapis.com",
		"replicapool.googleapis.com",
		"replicapoolupdater.googleapis.com",
		"resourceviews.googleapis.com",
		"runtimeconfig.googleapis.com",
		"servicecontrol.googleapis.com",
		"servicemanagement.googleapis.com",
		"sourcerepo.googleapis.com",
		"spanner.googleapis.com",
		"storage-api.googleapis.com",
		"storage-component.googleapis.com",
	}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccGoogleProjectAssociateServicesBasic_withBilling(services, pid, pname, org, billingId),
				Check: resource.ComposeTestCheckFunc(
					testProjectServicesMatch(services, pid),
				),
			},
		},
	})
}

func testAccGoogleProjectAssociateServicesBasic(services []string, pid, name, org string) string {
	return fmt.Sprintf(`
resource "google_project" "acceptance" {
  project_id = "%s"
  name = "%s"
  org_id = "%s"
}
resource "google_project_services" "acceptance" {
  project = "${google_project.acceptance.project_id}"
  services = [%s]
}
`, pid, name, org, testStringsToString(services))
}

func testAccGoogleProjectAssociateServicesBasic_withBilling(services []string, pid, name, org, billing string) string {
	return fmt.Sprintf(`
resource "google_project" "acceptance" {
  project_id = "%s"
  name = "%s"
  org_id = "%s"
  billing_account = "%s"
}
resource "google_project_services" "acceptance" {
  project = "${google_project.acceptance.project_id}"
  services = [%s]
}
`, pid, name, org, billing, testStringsToString(services))
}

func testProjectServicesMatch(services []string, pid string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		config := testAccProvider.Meta().(*Config)

		apiServices, err := getApiServices(pid, config)
		if err != nil {
			return fmt.Errorf("Error listing services for project %q: %v", pid, err)
		}

		sort.Strings(services)
		sort.Strings(apiServices)
		if !reflect.DeepEqual(services, apiServices) {
			return fmt.Errorf("Services in config (%v) do not exactly match services returned by API (%v)", services, apiServices)
		}

		return nil
	}
}

func testStringsToString(s []string) string {
	var b bytes.Buffer
	for i, v := range s {
		b.WriteString(fmt.Sprintf("\"%s\"", v))
		if i < len(s)-1 {
			b.WriteString(",")
		}
	}
	r := b.String()
	log.Printf("[DEBUG]: Converted list of strings to %s", r)
	return b.String()
}

func testManagedServicesToString(svcs []*servicemanagement.ManagedService) string {
	var b bytes.Buffer
	for _, s := range svcs {
		b.WriteString(s.ServiceName)
	}
	return b.String()
}
