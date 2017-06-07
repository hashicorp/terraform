provider "google" {
  // Provider settings to be provided via ENV variables
}

data "google_compute_zones" "available" {}

resource "random_id" "cluster_name" {
  byte_length = 10
}
resource "random_id" "username" {
  byte_length = 14
}
resource "random_id" "password" {
  byte_length = 16
}

resource "google_container_cluster" "primary" {
  name = "tf-acc-test-${random_id.cluster_name.hex}"
  zone = "${data.google_compute_zones.available.names[0]}"
  initial_node_count = 3

  additional_zones = [
    "${data.google_compute_zones.available.names[1]}"
  ]

  master_auth {
    username = "${random_id.username.hex}"
    password = "${random_id.password.hex}"
  }

  node_config {
    oauth_scopes = [
      "https://www.googleapis.com/auth/compute",
      "https://www.googleapis.com/auth/devstorage.read_only",
      "https://www.googleapis.com/auth/logging.write",
      "https://www.googleapis.com/auth/monitoring"
    ]
  }
}

output "zone" {
  value = "${data.google_compute_zones.available.names[0]}"
}

output "endpoint" {
  value = "${google_container_cluster.primary.endpoint}"
}

output "username" {
  value = "${google_container_cluster.primary.master_auth.0.username}"
}

output "password" {
  value = "${google_container_cluster.primary.master_auth.0.password}"
}

output "client_certificate_b64" {
  value = "${google_container_cluster.primary.master_auth.0.client_certificate}"
}

output "client_key_b64" {
  value = "${google_container_cluster.primary.master_auth.0.client_key}"
}

output "cluster_ca_certificate_b64" {
  value = "${google_container_cluster.primary.master_auth.0.cluster_ca_certificate}"
}
