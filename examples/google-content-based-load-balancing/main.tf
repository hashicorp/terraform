# https://cloud.google.com/compute/docs/load-balancing/http/content-based-example

provider "google" {
  region = "${var.region}"
  project = "${var.project_name}"
  credentials = "${file("${var.credentials_file_path}")}"
}

resource "google_compute_instance" "www" {
  name = "tf-www-compute"
  machine_type = "f1-micro"
  zone = "${var.region_zone}"
  tags = ["http-tag"]

  disk {
    image = "projects/debian-cloud/global/images/family/debian-8"
  }

  network_interface {
    network = "default"

    access_config {
      // Ephemeral IP
    }
  }

  metadata_startup_script = "${file("scripts/install-www.sh")}"

  service_account {
    scopes = ["https://www.googleapis.com/auth/compute.readonly"]
  }
}

resource "google_compute_instance" "www-video" {
  name = "tf-www-video-compute"
  machine_type = "f1-micro"
  zone = "${var.region_zone}"
  tags = ["http-tag"]

  disk {
    image = "projects/debian-cloud/global/images/family/debian-8"
  }

  network_interface {
    network = "default"

    access_config {
      // Ephemeral IP
    }
  }

  metadata_startup_script = "${file("scripts/install-video.sh")}"

  service_account {
    scopes = ["https://www.googleapis.com/auth/compute.readonly"]
  }
}

resource "google_compute_global_address" "external-address" {
  name = "tf-external-address"
}

resource "google_compute_instance_group" "www-resources" {
  name = "tf-www-resources"
  zone = "${var.region_zone}"

  instances = ["${google_compute_instance.www.self_link}"]

  named_port {
    name = "http"
    port = "80"
  }
}

resource "google_compute_instance_group" "video-resources" {
  name = "tf-video-resources"
  zone = "${var.region_zone}"

  instances = ["${google_compute_instance.www-video.self_link}"]

  named_port {
    name = "http"
    port = "80"
  }
}

resource "google_compute_health_check" "health-check" {
  name = "tf-health-check"

  http_health_check {
  }
}

resource "google_compute_backend_service" "www-service" {
  name = "tf-www-service"
  protocol = "HTTP"

  backend {
    group = "${google_compute_instance_group.www-resources.self_link}"
  }

  health_checks = ["${google_compute_health_check.health-check.self_link}"]
}

resource "google_compute_backend_service" "video-service" {
  name = "tf-video-service"
  protocol = "HTTP"

  backend {
    group = "${google_compute_instance_group.video-resources.self_link}"
  }

  health_checks = ["${google_compute_health_check.health-check.self_link}"]
}

resource "google_compute_url_map" "web-map" {
  name = "tf-web-map"
  default_service = "${google_compute_backend_service.www-service.self_link}"

  host_rule {
    hosts = ["*"]
    path_matcher = "tf-allpaths"
  }

  path_matcher {
    name = "tf-allpaths"
    default_service = "${google_compute_backend_service.www-service.self_link}"

    path_rule {
      paths = ["/video", "/video/*",]
      service = "${google_compute_backend_service.video-service.self_link}"
    }
  }
}

resource "google_compute_target_http_proxy" "http-lb-proxy" {
  name = "tf-http-lb-proxy"
  url_map = "${google_compute_url_map.web-map.self_link}"
}

resource "google_compute_global_forwarding_rule" "default" {
  name = "tf-http-content-gfr"
  target = "${google_compute_target_http_proxy.http-lb-proxy.self_link}"
  ip_address = "${google_compute_global_address.external-address.address}"
  port_range = "80"
}

resource "google_compute_firewall" "default" {
  name = "tf-www-firewall-allow-internal-only"
  network = "default"

  allow {
    protocol = "tcp"
    ports = ["80"]
  }

  source_ranges = ["130.211.0.0/22", "35.191.0.0/16"]
  target_tags = ["http-tag"]
}
