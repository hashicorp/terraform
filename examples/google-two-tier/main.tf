# See https://cloud.google.com/compute/docs/load-balancing/network/example

provider "google" {
  region = "${var.region}"
  project = "${var.project_name}"
  account_file = "${file(var.account_file_path)}"
}

resource "google_compute_http_health_check" "default" {
  name = "tf-www-basic-check"
  request_path = "/"
  check_interval_sec = 1
  healthy_threshold = 1
  unhealthy_threshold = 10
  timeout_sec = 1
}

resource "google_compute_target_pool" "default" {
  name = "tf-www-target-pool"
  instances = ["${google_compute_instance.www.*.self_link}"]
  health_checks = ["${google_compute_http_health_check.default.name}"]
}

resource "google_compute_forwarding_rule" "default" {
  name = "tf-www-forwarding-rule"
  target = "${google_compute_target_pool.default.self_link}"
  port_range = "80"
}

resource "google_compute_instance" "www" {
  count = 3

  name = "tf-www-${count.index}"
  machine_type = "n1-standard-1"
  zone = "${var.region_zone}"
  tags = ["www-node"]

  disk {
    image = "ubuntu-os-cloud/ubuntu-1204-precise-v20150625"
  }

  network_interface {
    network = "default"
    access_config {
        # Ephemeral
    }
  }

  metadata {
    sshKeys = "ubuntu:${file("~/.ssh/gcloud_id_rsa.pub")}"
    startup-script = <<SCRIPT
apt-get -y update
apt-get -y install nginx
HOSTNAME=$(hostname | tr -d "\n")
IP=$(curl -s -H "Metadata-Flavor:Google" http://metadata/computeMetadata/v1/instance/network-interfaces/0/ip)
echo "Welcome to ${count.index} - $HOSTNAME ($IP)" > /usr/share/nginx/www/index.html
service nginx start
SCRIPT
  }

  service_account {
    scopes = ["https://www.googleapis.com/auth/compute.readonly"]
  }
}

resource "google_compute_firewall" "default" {
  name = "tf-www-firewall"
  network = "default"

  allow {
    protocol = "tcp"
    ports = ["80"]
  }

  source_ranges = ["0.0.0.0/0"]
  target_tags = ["www-node"]
}
