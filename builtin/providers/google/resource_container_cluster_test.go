package google

import (
	"fmt"
	"testing"

	"strconv"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccContainerCluster_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckContainerClusterDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccContainerCluster_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckContainerCluster(
						"google_container_cluster.primary"),
				),
			},
		},
	})
}

func TestAccContainerCluster_withAdditionalZones(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckContainerClusterDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccContainerCluster_withAdditionalZones,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckContainerCluster(
						"google_container_cluster.with_additional_zones"),
				),
			},
		},
	})
}

func TestAccContainerCluster_withVersion(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckContainerClusterDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccContainerCluster_withVersion,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckContainerCluster(
						"google_container_cluster.with_version"),
				),
			},
		},
	})
}

func TestAccContainerCluster_withNodeConfig(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckContainerClusterDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccContainerCluster_withNodeConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckContainerCluster(
						"google_container_cluster.with_node_config"),
				),
			},
		},
	})
}

func TestAccContainerCluster_withNodeConfigScopeAlias(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckContainerClusterDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccContainerCluster_withNodeConfigScopeAlias,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckContainerCluster(
						"google_container_cluster.with_node_config_scope_alias"),
				),
			},
		},
	})
}

func TestAccContainerCluster_network(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckContainerClusterDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccContainerCluster_networkRef,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckContainerCluster(
						"google_container_cluster.with_net_ref_by_url"),
					testAccCheckContainerCluster(
						"google_container_cluster.with_net_ref_by_name"),
				),
			},
		},
	})
}

func TestAccContainerCluster_backend(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckContainerClusterDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccContainerCluster_backendRef,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckContainerClusterExists(
						"google_container_cluster.primary"),
				),
			},
		},
	})
}

func testAccCheckContainerClusterDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "google_container_cluster" {
			continue
		}

		attributes := rs.Primary.Attributes
		_, err := config.clientContainer.Projects.Zones.Clusters.Get(
			config.Project, attributes["zone"], attributes["name"]).Do()
		if err == nil {
			return fmt.Errorf("Cluster still exists")
		}
	}

	return nil
}

func testAccCheckContainerCluster(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)

		attributes := rs.Primary.Attributes
		cluster, err := config.clientContainer.Projects.Zones.Clusters.Get(
			config.Project, attributes["zone"], attributes["name"]).Do()
		if err != nil {
			return err
		}

		if cluster.Name != attributes["name"] {
			return fmt.Errorf("Cluster not found")
		}

		if c := checkMatch(attributes, "initial_node_count", strconv.FormatInt(cluster.InitialNodeCount, 10)); c != "" {
			return fmt.Errorf(c)
		}

		if c := checkMatch(attributes, "master_auth.0.client_certificate", cluster.MasterAuth.ClientCertificate); c != "" {
			return fmt.Errorf(c)
		}

		if c := checkMatch(attributes, "master_auth.0.client_key", cluster.MasterAuth.ClientKey); c != "" {
			return fmt.Errorf(c)
		}

		if c := checkMatch(attributes, "master_auth.0.cluster_ca_certificate", cluster.MasterAuth.ClusterCaCertificate); c != "" {
			return fmt.Errorf(c)
		}

		if c := checkMatch(attributes, "master_auth.0.password", cluster.MasterAuth.Password); c != "" {
			return fmt.Errorf(c)
		}

		if c := checkMatch(attributes, "master_auth.0.username", cluster.MasterAuth.Username); c != "" {
			return fmt.Errorf(c)
		}

		if c := checkMatch(attributes, "zone", cluster.Zone); c != "" {
			return fmt.Errorf(c)
		}

		additionalZones := []string{}
		for _, location := range cluster.Locations {
			if location != cluster.Zone {
				additionalZones = append(additionalZones, location)
			}
		}

		if c := checkListMatch(attributes, "additional_zones", additionalZones); c != "" {
			return fmt.Errorf(c)
		}

		if c := checkMatch(attributes, "cluster_ipv4_cidr", cluster.ClusterIpv4Cidr); c != "" {
			return fmt.Errorf(c)
		}

		if c := checkMatch(attributes, "description", cluster.Description); c != "" {
			return fmt.Errorf(c)
		}

		if c := checkMatch(attributes, "endpoint", cluster.Endpoint); c != "" {
			return fmt.Errorf(c)
		}

		if c := checkListMatch(attributes, "instance_group_urls", cluster.InstanceGroupUrls); c != "" {
			return fmt.Errorf("%s,\nIGU[0]: %s", c, attributes["instance_group_urls.0"])
		}

		if c := checkMatch(attributes, "logging_service", cluster.LoggingService); c != "" {
			return fmt.Errorf(c)
		}

		if c := checkMatch(attributes, "monitoring_service", cluster.MonitoringService); c != "" {
			return fmt.Errorf(c)
		}

		// TODO(danawillow): Add this back in. Currently this field is saved via the config instead of from the API response,
		// and the config may contain the network name or self_link, whereas the API only returns the self_link.
		// if c := checkMatch(attributes, "network", cluster.Network, "network"); c != "" {
		// 	return fmt.Errorf(c)
		// }

		if c := checkMatch(attributes, "subnetwork", cluster.Subnetwork); c != "" {
			return fmt.Errorf(c)
		}

		// AddonsConfig is neither Required or Computed, so the API may return nil for it
		if cluster.AddonsConfig != nil {
			if cluster.AddonsConfig.HttpLoadBalancing != nil {
				if c := checkMatch(attributes, "addons_config.0.http_load_balancing.0.disabled", strconv.FormatBool(cluster.AddonsConfig.HttpLoadBalancing.Disabled)); c != "" {
					return fmt.Errorf(c)
				}
			}

			if cluster.AddonsConfig.HorizontalPodAutoscaling != nil {
				if c := checkMatch(attributes, "addons_config.0.horizontal_pod_autoscaling.0.disabled", strconv.FormatBool(cluster.AddonsConfig.HorizontalPodAutoscaling.Disabled)); c != "" {
					return fmt.Errorf(c)
				}
			}
		}

		if c := checkMatch(attributes, "node_config.0.machine_type", cluster.NodeConfig.MachineType); c != "" {
			return fmt.Errorf(c)
		}

		if c := checkMatch(attributes, "node_config.0.disk_size_gb", strconv.FormatInt(cluster.NodeConfig.DiskSizeGb, 10)); c != "" {
			return fmt.Errorf(c)
		}

		if c := checkListMatch(attributes, "node_config.0.oauth_scopes", cluster.NodeConfig.OauthScopes); c != "" {
			return fmt.Errorf(c)
		}

		if c := checkMatch(attributes, "node_version", cluster.CurrentNodeVersion); c != "" {
			return fmt.Errorf(c)
		}
		return nil
	}
}

func checkMatch(attributes map[string]string, attr string, gcp interface{}) string {
	tf := attributes[attr]
	if tf != gcp {
		return fmt.Sprintf("Cluster has mismatched %s.\nTF State: %+v\nGCP State: %+v", attr, tf, gcp)
	}
	return ""
}

func checkListMatch(attributes map[string]string, attr string, gcpList []string) string {
	num, err := strconv.Atoi(attributes[attr+".#"])
	if err != nil {
		return fmt.Sprintf("error in number conversion for attribute %s", attr)
	}
	if num != len(gcpList) {
		return fmt.Sprintf("Cluster has mismatched %s size.\nTF Size: %d\nGCP Size: %d", attr, num, len(gcpList))
	}

	for i, gcp := range gcpList {
		if tf := attributes[fmt.Sprintf("%s.%d", attr, i)]; tf != gcp {
			return fmt.Sprintf("Cluster has mismatched %s[%d].\nTF State: %+v\nGCP State: %+v", attr, i, tf, gcp)
		}
	}

	return ""
}

var testAccContainerCluster_basic = fmt.Sprintf(`
resource "google_container_cluster" "primary" {
	name = "cluster-test-%s"
	zone = "us-central1-a"
	initial_node_count = 3

	master_auth {
		username = "mr.yoda"
		password = "adoy.rm"
	}
}`, acctest.RandString(10))

var testAccContainerCluster_withAdditionalZones = fmt.Sprintf(`
resource "google_container_cluster" "with_additional_zones" {
	name = "cluster-test-%s"
	zone = "us-central1-a"
	initial_node_count = 1

	additional_zones = [
		"us-central1-b",
		"us-central1-c"
	]

	master_auth {
		username = "mr.yoda"
		password = "adoy.rm"
	}
}`, acctest.RandString(10))

var testAccContainerCluster_withVersion = fmt.Sprintf(`
resource "google_container_cluster" "with_version" {
	name = "cluster-test-%s"
	zone = "us-central1-a"
	node_version = "1.5.2"
	initial_node_count = 1

	master_auth {
		username = "mr.yoda"
		password = "adoy.rm"
	}
}`, acctest.RandString(10))

var testAccContainerCluster_withNodeConfig = fmt.Sprintf(`
resource "google_container_cluster" "with_node_config" {
	name = "cluster-test-%s"
	zone = "us-central1-f"
	initial_node_count = 1

	master_auth {
		username = "mr.yoda"
		password = "adoy.rm"
	}

	node_config {
		machine_type = "g1-small"
		disk_size_gb = 15
		oauth_scopes = [
			"https://www.googleapis.com/auth/compute",
			"https://www.googleapis.com/auth/devstorage.read_only",
			"https://www.googleapis.com/auth/logging.write",
			"https://www.googleapis.com/auth/monitoring"
		]
	}
}`, acctest.RandString(10))

var testAccContainerCluster_withNodeConfigScopeAlias = fmt.Sprintf(`
resource "google_container_cluster" "with_node_config_scope_alias" {
	name = "cluster-test-%s"
	zone = "us-central1-f"
	initial_node_count = 1

	master_auth {
		username = "mr.yoda"
		password = "adoy.rm"
	}

	node_config {
		machine_type = "g1-small"
		disk_size_gb = 15
		oauth_scopes = [ "compute-rw", "storage-ro", "logging-write", "monitoring" ]
	}
}`, acctest.RandString(10))

var testAccContainerCluster_networkRef = fmt.Sprintf(`
resource "google_compute_network" "container_network" {
	name = "container-net-%s"
	auto_create_subnetworks = true
}

resource "google_container_cluster" "with_net_ref_by_url" {
	name = "cluster-test-%s"
	zone = "us-central1-a"
	initial_node_count = 1

	master_auth {
		username = "mr.yoda"
		password = "adoy.rm"
	}

	network = "${google_compute_network.container_network.self_link}"
}

resource "google_container_cluster" "with_net_ref_by_name" {
	name = "cluster-test-%s"
	zone = "us-central1-a"
	initial_node_count = 1

	master_auth {
		username = "mr.yoda"
		password = "adoy.rm"
	}

	network = "${google_compute_network.container_network.name}"
}`, acctest.RandString(10), acctest.RandString(10), acctest.RandString(10))

var testAccContainerCluster_backendRef = fmt.Sprintf(`
resource "google_compute_backend_service" "my-backend-service" {
  name      = "terraform-test-%s"
  port_name = "http"
  protocol  = "HTTP"

  backend {
    group = "${element(google_container_cluster.primary.instance_group_urls, 1)}"
  }

  health_checks = ["${google_compute_http_health_check.default.self_link}"]
}

resource "google_compute_http_health_check" "default" {
  name               = "terraform-test-%s"
  request_path       = "/"
  check_interval_sec = 1
  timeout_sec        = 1
}

resource "google_container_cluster" "primary" {
  name               = "terraform-test-%s"
  zone               = "us-central1-a"
  initial_node_count = 3

  additional_zones = [
    "us-central1-b",
    "us-central1-c",
  ]

  master_auth {
    username = "mr.yoda"
    password = "adoy.rm"
  }

  node_config {
    oauth_scopes = [
      "https://www.googleapis.com/auth/compute",
      "https://www.googleapis.com/auth/devstorage.read_only",
      "https://www.googleapis.com/auth/logging.write",
      "https://www.googleapis.com/auth/monitoring",
    ]
  }
}
`, acctest.RandString(10), acctest.RandString(10), acctest.RandString(10))
