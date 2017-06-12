package google

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"google.golang.org/api/compute/v1"
)

func TestAccComputeTargetHttpsProxy_basic(t *testing.T) {
	var targetHttpsProxy compute.TargetHttpsProxy
	var resourceName = fmt.Sprintf("httpsproxy-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeTargetHttpsProxyDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeTargetHttpsProxy_basic(resourceName, 0, 0),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeTargetHttpsProxyExists(
						"google_compute_target_https_proxy.basic", &targetHttpsProxy),
				),
			},
		},
	})
}

func TestAccComputeTargetHttpsProxy_update(t *testing.T) {
	var targetHttpsProxy compute.TargetHttpsProxy
	var resourceName = fmt.Sprintf("httpsproxy-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeTargetHttpsProxyDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeTargetHttpsProxy_basic(resourceName, 0, 0),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeTargetHttpsProxyExists(
						"google_compute_target_https_proxy.basic", &targetHttpsProxy),
					testAccCheckComputeTargetHttpsProxyHasCorrectUrlMap(
						&targetHttpsProxy, fmt.Sprintf("%s-%d", resourceName, 0)),
					testAccCheckComputeTargetHttpsProxyHasCorrectSslCertificates(
						&targetHttpsProxy, []string{fmt.Sprintf("%s-%d", resourceName, 0)}),
				),
			},

			// Update urlMap: 0 -> 1
			resource.TestStep{
				Config: testAccComputeTargetHttpsProxy_basic(resourceName, 1, 0),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeTargetHttpsProxyExists(
						"google_compute_target_https_proxy.basic", &targetHttpsProxy),
					testAccCheckComputeTargetHttpsProxyHasCorrectUrlMap(
						&targetHttpsProxy, fmt.Sprintf("%s-%d", resourceName, 1)),
					testAccCheckComputeTargetHttpsProxyHasCorrectSslCertificates(
						&targetHttpsProxy, []string{fmt.Sprintf("%s-%d", resourceName, 0)}),
				),
			},

			// Update sslCertificate: 0 -> 1
			resource.TestStep{
				Config: testAccComputeTargetHttpsProxy_basic(resourceName, 1, 1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeTargetHttpsProxyExists(
						"google_compute_target_https_proxy.basic", &targetHttpsProxy),
					testAccCheckComputeTargetHttpsProxyHasCorrectUrlMap(
						&targetHttpsProxy, fmt.Sprintf("%s-%d", resourceName, 1)),
					testAccCheckComputeTargetHttpsProxyHasCorrectSslCertificates(
						&targetHttpsProxy, []string{fmt.Sprintf("%s-%d", resourceName, 1)}),
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

func testAccCheckComputeTargetHttpsProxyExists(n string, targetHttpsProxy *compute.TargetHttpsProxy) resource.TestCheckFunc {
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

		*targetHttpsProxy = *found

		return nil
	}
}

func testAccCheckComputeTargetHttpsProxyHasCorrectUrlMap(targetHttpsProxy *compute.TargetHttpsProxy, urlMapName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		config := testAccProvider.Meta().(*Config)
		urlMap := fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/global/urlMaps/%s", config.Project, urlMapName)
		if targetHttpsProxy.UrlMap != urlMap {
			return fmt.Errorf("Invalid URL map: got '%s', while expected '%s'", targetHttpsProxy.UrlMap, urlMap)
		}

		return nil
	}
}

func testAccCheckComputeTargetHttpsProxyHasCorrectSslCertificates(targetHttpsProxy *compute.TargetHttpsProxy, sslCertificateNames []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if len(sslCertificateNames) > 0 && targetHttpsProxy.SslCertificates == nil {
			return fmt.Errorf("No SSL Certificates are set")
		}

		config := testAccProvider.Meta().(*Config)
		certsCounter := map[string]int{}
		for _, cert := range targetHttpsProxy.SslCertificates {
			certsCounter[cert] += 1
		}
		for _, certName := range sslCertificateNames {
			cert := fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/global/sslCertificates/%s", config.Project, certName)
			certsCounter[cert] -= 1
		}
		unwantedCerts := []string{}
		lostCerts := []string{}
		for cert, cnt := range certsCounter {
			if cnt > 0 {
				unwantedCerts = append(unwantedCerts, cert)
			} else if cnt < 0 {
				lostCerts = append(lostCerts, cert)
			}
		}
		if len(unwantedCerts) > 0 || len(lostCerts) > 0 {
			return fmt.Errorf("SSL certificates mismatch: unwanted=%v, lost=%v", unwantedCerts, lostCerts)
		}

		return nil
	}
}

func testAccComputeTargetHttpsProxy_basic(name string, urlMapIndex int, sslCertificateIndex int) string {
	return fmt.Sprintf(`
	resource "google_compute_http_health_check" "basic" {
		name = "%[1]s"
	}

	resource "google_compute_backend_service" "basic" {
		name = "%[1]s"
		health_checks = ["${google_compute_http_health_check.basic.self_link}"]
	}

	resource "google_compute_url_map" "basic" {
		count = 2
		name = "%[1]s-${count.index}"
		default_service = "${google_compute_backend_service.basic.self_link}"
	}

	resource "google_compute_ssl_certificate" "basic" {
		count = 2
		name = "%[1]s-${count.index}"
		private_key = "${file("test-fixtures/ssl_cert/test.key")}"
		certificate = "${file("test-fixtures/ssl_cert/test.crt")}"
	}

	resource "google_compute_target_https_proxy" "basic" {
		description = "Resource created for Terraform acceptance testing"
		name = "%[1]s"
		url_map = "${google_compute_url_map.basic.%[2]d.self_link}"
		ssl_certificates = ["${google_compute_ssl_certificate.basic.%[3]d.self_link}"]
	}`, name, urlMapIndex, sslCertificateIndex)
}
