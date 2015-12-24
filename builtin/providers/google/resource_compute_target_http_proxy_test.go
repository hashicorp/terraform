package google

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccComputeTargetHttpProxy_basic(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeTargetHttpProxyDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeTargetHttpProxy_basic1,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeTargetHttpProxyExists(
						"google_compute_target_http_proxy.foobar"),
				),
			},
		},
	})
}

func TestAccComputeTargetHttpProxy_update(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeTargetHttpProxyDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeTargetHttpProxy_basic1,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeTargetHttpProxyExists(
						"google_compute_target_http_proxy.foobar"),
				),
			},

			resource.TestStep{
				Config: testAccComputeTargetHttpProxy_basic2,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeTargetHttpProxyExists(
						"google_compute_target_http_proxy.foobar"),
				),
			},
		},
	})
}

func testAccCheckComputeTargetHttpProxyDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "google_compute_target_http_proxy" {
			continue
		}

		_, err := config.clientCompute.TargetHttpProxies.Get(
			config.Project, rs.Primary.ID).Do()
		if err == nil {
			return fmt.Errorf("TargetHttpProxy still exists")
		}
	}

	return nil
}

func testAccCheckComputeTargetHttpProxyExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)

		found, err := config.clientCompute.TargetHttpProxies.Get(
			config.Project, rs.Primary.ID).Do()
		if err != nil {
			return err
		}

		if found.Name != rs.Primary.ID {
			return fmt.Errorf("TargetHttpProxy not found")
		}

		return nil
	}
}

const testAccComputeTargetHttpProxy_basic1 = `
resource "google_compute_target_http_proxy" "foobar" {
	description = "Resource created for Terraform acceptance testing"
	name = "terraform-test"
	url_map = "${google_compute_url_map.foobar1.self_link}"
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

resource "google_compute_url_map" "foobar1" {
	name = "myurlmap1"
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

resource "google_compute_url_map" "foobar2" {
	name = "myurlmap2"
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
`

const testAccComputeTargetHttpProxy_basic2 = `
resource "google_compute_target_http_proxy" "foobar" {
	description = "Resource created for Terraform acceptance testing"
	name = "terraform-test"
	url_map = "${google_compute_url_map.foobar2.self_link}"
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

resource "google_compute_url_map" "foobar1" {
	name = "myurlmap1"
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

resource "google_compute_url_map" "foobar2" {
	name = "myurlmap2"
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
`
