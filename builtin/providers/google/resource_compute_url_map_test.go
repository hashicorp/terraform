package google

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccComputeUrlMap_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeUrlMapDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeUrlMap_basic1,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeUrlMapExists(
						"google_compute_url_map.foobar"),
				),
			},
		},
	})
}

func TestAccComputeUrlMap_update_path_matcher(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeUrlMapDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeUrlMap_basic1,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeUrlMapExists(
						"google_compute_url_map.foobar"),
				),
			},

			resource.TestStep{
				Config: testAccComputeUrlMap_basic2,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeUrlMapExists(
						"google_compute_url_map.foobar"),
				),
			},
		},
	})
}

func TestAccComputeUrlMap_advanced(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeUrlMapDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeUrlMap_advanced1,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeUrlMapExists(
						"google_compute_url_map.foobar"),
				),
			},

			resource.TestStep{
				Config: testAccComputeUrlMap_advanced2,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeUrlMapExists(
						"google_compute_url_map.foobar"),
				),
			},
		},
	})
}

func testAccCheckComputeUrlMapDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "google_compute_url_map" {
			continue
		}

		_, err := config.clientCompute.UrlMaps.Get(
			config.Project, rs.Primary.ID).Do()
		if err == nil {
			return fmt.Errorf("Url map still exists")
		}
	}

	return nil
}

func testAccCheckComputeUrlMapExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)

		found, err := config.clientCompute.UrlMaps.Get(
			config.Project, rs.Primary.ID).Do()
		if err != nil {
			return err
		}

		if found.Name != rs.Primary.ID {
			return fmt.Errorf("Url map not found")
		}
		return nil
	}
}

var testAccComputeUrlMap_basic1 = fmt.Sprintf(`
resource "google_compute_backend_service" "foobar" {
    name = "urlmap-test-%s"
    health_checks = ["${google_compute_http_health_check.zero.self_link}"]
}

resource "google_compute_http_health_check" "zero" {
    name = "urlmap-test-%s"
    request_path = "/"
    check_interval_sec = 1
    timeout_sec = 1
}

resource "google_compute_url_map" "foobar" {
    name = "urlmap-test-%s"
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
`, acctest.RandString(10), acctest.RandString(10), acctest.RandString(10))

var testAccComputeUrlMap_basic2 = fmt.Sprintf(`
resource "google_compute_backend_service" "foobar" {
    name = "urlmap-test-%s"
    health_checks = ["${google_compute_http_health_check.zero.self_link}"]
}

resource "google_compute_http_health_check" "zero" {
    name = "urlmap-test-%s"
    request_path = "/"
    check_interval_sec = 1
    timeout_sec = 1
}

resource "google_compute_url_map" "foobar" {
    name = "urlmap-test-%s"
	default_service = "${google_compute_backend_service.foobar.self_link}"

    host_rule {
        hosts = ["mysite.com", "myothersite.com"]
        path_matcher = "blip"
    }

    path_matcher {
        default_service = "${google_compute_backend_service.foobar.self_link}"
        name = "blip"
        path_rule {
            paths = ["/*", "/home"]
            service = "${google_compute_backend_service.foobar.self_link}"
        }
    }

	test {
		host = "mysite.com"
		path = "/*"
		service = "${google_compute_backend_service.foobar.self_link}"
	}
}
`, acctest.RandString(10), acctest.RandString(10), acctest.RandString(10))

var testAccComputeUrlMap_advanced1 = fmt.Sprintf(`
resource "google_compute_backend_service" "foobar" {
    name = "urlmap-test-%s"
    health_checks = ["${google_compute_http_health_check.zero.self_link}"]
}

resource "google_compute_http_health_check" "zero" {
    name = "urlmap-test-%s"
    request_path = "/"
    check_interval_sec = 1
    timeout_sec = 1
}

resource "google_compute_url_map" "foobar" {
    name = "urlmap-test-%s"
	default_service = "${google_compute_backend_service.foobar.self_link}"

    host_rule {
        hosts = ["mysite.com", "myothersite.com"]
        path_matcher = "blop"
    }

    host_rule {
        hosts = ["myfavoritesite.com"]
        path_matcher = "blip"
    }

    path_matcher {
        default_service = "${google_compute_backend_service.foobar.self_link}"
        name = "blop"
        path_rule {
            paths = ["/*", "/home"]
            service = "${google_compute_backend_service.foobar.self_link}"
        }
    }

    path_matcher {
        default_service = "${google_compute_backend_service.foobar.self_link}"
        name = "blip"
        path_rule {
            paths = ["/*", "/home"]
            service = "${google_compute_backend_service.foobar.self_link}"
        }
    }
}
`, acctest.RandString(10), acctest.RandString(10), acctest.RandString(10))

var testAccComputeUrlMap_advanced2 = fmt.Sprintf(`
resource "google_compute_backend_service" "foobar" {
    name = "urlmap-test-%s"
    health_checks = ["${google_compute_http_health_check.zero.self_link}"]
}

resource "google_compute_http_health_check" "zero" {
    name = "urlmap-test-%s"
    request_path = "/"
    check_interval_sec = 1
    timeout_sec = 1
}

resource "google_compute_url_map" "foobar" {
    name = "urlmap-test-%s"
	default_service = "${google_compute_backend_service.foobar.self_link}"

    host_rule {
        hosts = ["mysite.com", "myothersite.com"]
        path_matcher = "blep"
    }

    host_rule {
        hosts = ["myfavoritesite.com"]
        path_matcher = "blip"
    }

    host_rule {
        hosts = ["myleastfavoritesite.com"]
        path_matcher = "blub"
    }

    path_matcher {
        default_service = "${google_compute_backend_service.foobar.self_link}"
        name = "blep"
        path_rule {
            paths = ["/home"]
            service = "${google_compute_backend_service.foobar.self_link}"
        }

        path_rule {
            paths = ["/login"]
            service = "${google_compute_backend_service.foobar.self_link}"
        }
    }

    path_matcher {
        default_service = "${google_compute_backend_service.foobar.self_link}"
        name = "blub"
        path_rule {
            paths = ["/*", "/blub"]
            service = "${google_compute_backend_service.foobar.self_link}"
        }
    }

    path_matcher {
        default_service = "${google_compute_backend_service.foobar.self_link}"
        name = "blip"
        path_rule {
            paths = ["/*", "/home"]
            service = "${google_compute_backend_service.foobar.self_link}"
        }
    }
}
`, acctest.RandString(10), acctest.RandString(10), acctest.RandString(10))
