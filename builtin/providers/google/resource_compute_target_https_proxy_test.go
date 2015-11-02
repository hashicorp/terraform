package google

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccComputeTargetHttpsProxy_basic(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeTargetHttpsProxyDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeTargetHttpsProxy_basic1,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeTargetHttpsProxyExists(
						"google_compute_target_https_proxy.foobar"),
				),
			},
		},
	})
}

func TestAccComputeTargetHttpsProxy_update(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeTargetHttpsProxyDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeTargetHttpsProxy_basic1,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeTargetHttpsProxyExists(
						"google_compute_target_https_proxy.foobar"),
				),
			},

			resource.TestStep{
				Config: testAccComputeTargetHttpsProxy_basic2,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeTargetHttpsProxyExists(
						"google_compute_target_https_proxy.foobar"),
				),
			},
		},
	})
}

func testAccCheckComputeTargetHttpsProxyDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "google_compute_target_https_proxy" {
			continue
		}

		_, err := config.clientCompute.TargetHttpsProxies.Get(
			config.Project, rs.Primary.ID).Do()
		if err == nil {
			return fmt.Errorf("TargetHttpsProxy still exists")
		}
	}

	return nil
}

func testAccCheckComputeTargetHttpsProxyExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)

		found, err := config.clientCompute.TargetHttpsProxies.Get(
			config.Project, rs.Primary.ID).Do()
		if err != nil {
			return err
		}

		if found.Name != rs.Primary.ID {
			return fmt.Errorf("TargetHttpsProxy not found")
		}

		return nil
	}
}

const testAccComputeTargetHttpsProxy_basic1 = `
resource "google_compute_target_https_proxy" "foobar" {
	description = "Resource created for Terraform acceptance testing"
	name = "terraform-test"
	url_map = "${google_compute_url_map.foobar.self_link}"
	ssl_certificates = ["${google_compute_ssl_certificate.foobar1.self_link}"]
}

resource "google_compute_backend_service" "foobar" {
	name = "service"
	health_checks = ["${google_compute_http_health_check.zero.self_link}"]
}

resource "google_compute_http_health_check" "zero" {
	name = "tf-test-zero"
	request_path = "/"
	check_interval_sec = 1
	timeout_sec = 1
}

resource "google_compute_url_map" "foobar" {
	name = "myurlmap"
	default_service = "${google_compute_backend_service.foobar.self_link}"
	host_rule {
		hosts = ["mysite.com", "myothersite.com"]
		path_matcher = "boop"
	}
	path_matcher {
		default_service = "${google_compute_backend_service.foobar.self_link}"
		name = "boop"
		path_rule {
			paths = ["/*"]
			service = "${google_compute_backend_service.foobar.self_link}"
		}
	}
	test {
		host = "mysite.com"
		path = "/*"
		service = "${google_compute_backend_service.foobar.self_link}"
	}
}

resource "google_compute_ssl_certificate" "foobar1" {
	name = "terraform-test1"
	description = "very descriptive"
	private_key = "${file("~/cert/example.key")}"
	certificate = "${file("~/cert/example.crt")}"
}

resource "google_compute_ssl_certificate" "foobar2" {
	name = "terraform-test2"
	description = "very descriptive"
	private_key = "${file("~/cert/example.key")}"
	certificate = "${file("~/cert/example.crt")}"
}
`

const testAccComputeTargetHttpsProxy_basic2 = `
resource "google_compute_target_https_proxy" "foobar" {
	description = "Resource created for Terraform acceptance testing"
	name = "terraform-test"
	url_map = "${google_compute_url_map.foobar.self_link}"
	ssl_certificates = ["${google_compute_ssl_certificate.foobar1.self_link}"]
}

resource "google_compute_backend_service" "foobar" {
	name = "service"
	health_checks = ["${google_compute_http_health_check.zero.self_link}"]
}

resource "google_compute_http_health_check" "zero" {
	name = "tf-test-zero"
	request_path = "/"
	check_interval_sec = 1
	timeout_sec = 1
}

resource "google_compute_url_map" "foobar" {
	name = "myurlmap"
	default_service = "${google_compute_backend_service.foobar.self_link}"
	host_rule {
		hosts = ["mysite.com", "myothersite.com"]
		path_matcher = "boop"
	}
	path_matcher {
		default_service = "${google_compute_backend_service.foobar.self_link}"
		name = "boop"
		path_rule {
			paths = ["/*"]
			service = "${google_compute_backend_service.foobar.self_link}"
		}
	}
	test {
		host = "mysite.com"
		path = "/*"
		service = "${google_compute_backend_service.foobar.self_link}"
	}
}

resource "google_compute_ssl_certificate" "foobar1" {
	name = "terraform-test1"
	description = "very descriptive"
	private_key = "${file("~/cert/example.key")}"
	certificate = "${file("~/cert/example.crt")}"
}

resource "google_compute_ssl_certificate" "foobar2" {
	name = "terraform-test2"
	description = "very descriptive"
	private_key = "${file("~/cert/example.key")}"
	certificate = "${file("~/cert/example.crt")}"
}
`
