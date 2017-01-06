provider "google" {
	region      = "${var.region}"
	project     = "${var.project_name}"
}

resource "google_compute_network" "my-custom-network" {
	name = "my-custom-network"
}

resource "google_compute_subnetwork" "my-custom-subnet" {
	name          = "my-custom-subnet"
	ip_cidr_range = "10.128.0.0/20"
	network       = "${google_compute_network.my-custom-network.self_link}"
	region        = "${var.region}"
}

resource "google_compute_firewall" "allow-all-internal" {
	name    = "allow-all-10-128-0-0-20"
	network = "${google_compute_network.my-custom-network.name}"

	allow {
		protocol = "tcp"
	}

	allow {
		protocol = "udp"
	}

	allow {
		protocol = "icmp"
	}

	source_ranges = ["10.128.0.0/20"]
}

resource "google_compute_firewall" "allow-ssh-rdp-icmp" {
	name    = "allow-tcp22-tcp3389-icmp"
	network = "${google_compute_network.my-custom-network.name}"

	allow {
		protocol = "tcp"
		ports    = ["22", "3389",]
	}

	allow {
		protocol = "icmp"
	}
}

resource "google_compute_instance" "ilb-instance-1" {
	name         = "ilb-instance-1"
	machine_type = "n1-standard-1"
	zone         = "${var.region_zone}"

	tags = ["int-lb"]

	disk {
		image = "debian-cloud/debian-8"
	}

	network_interface {
		subnetwork = "${google_compute_subnetwork.my-custom-subnet.name}"
		access_config {
			// Ephemeral IP
		}
	}

	service_account {
    	scopes = ["compute-rw"]
  	}

	metadata_startup_script = "${file("startup.sh")}"
}

resource "google_compute_instance" "ilb-instance-2" {
	name         = "ilb-instance-2"
	machine_type = "n1-standard-1"
	zone         = "${var.region_zone}"

	tags = ["int-lb"]

	disk {
		image = "debian-cloud/debian-8"
	}

	network_interface {
		subnetwork = "${google_compute_subnetwork.my-custom-subnet.name}"
		access_config {
			// Ephemeral IP
		}
	}

	service_account {
    	scopes = ["compute-rw"]
  	}

	metadata_startup_script = "${file("startup.sh")}"
}

resource "google_compute_instance" "ilb-instance-3" {
	name         = "ilb-instance-3"
	machine_type = "n1-standard-1"
	zone         = "${var.region_zone_2}"

	tags = ["int-lb"]

	disk {
		image = "debian-cloud/debian-8"
	}

	network_interface {
		subnetwork = "${google_compute_subnetwork.my-custom-subnet.name}"
		access_config {
			// Ephemeral IP
		}
	}

	service_account {
    	scopes = ["compute-rw"]
  	}

	metadata_startup_script = "${file("startup.sh")}"
}

resource "google_compute_instance" "ilb-instance-4" {
	name         = "ilb-instance-4"
	machine_type = "n1-standard-1"
	zone         = "${var.region_zone_2}"

	tags = ["int-lb"]

	disk {
		image = "debian-cloud/debian-8"
	}

	network_interface {
		subnetwork = "${google_compute_subnetwork.my-custom-subnet.name}"
		access_config {
			// Ephemeral IP
		}
	}

	service_account {
    	scopes = ["compute-rw"]
  	}

	metadata_startup_script = "${file("startup.sh")}"
}

resource "google_compute_instance_group" "us-ig1" {
	name        = "us-ig1"

	instances = [
		"${google_compute_instance.ilb-instance-1.self_link}",
		"${google_compute_instance.ilb-instance-2.self_link}"
	]

	zone = "${var.region_zone}"
}

resource "google_compute_instance_group" "us-ig2" {
	name        = "us-ig2"

	instances = [
		"${google_compute_instance.ilb-instance-3.self_link}",
		"${google_compute_instance.ilb-instance-4.self_link}"
	]

	zone = "${var.region_zone_2}"
}

resource "google_compute_health_check" "my-tcp-health-check" {
	name = "my-tcp-health-check"

	tcp_health_check {
		port = "80"
	}
}

resource "google_compute_region_backend_service" "my-int-lb" {
	name                  = "my-int-lb"
	health_checks         = ["${google_compute_health_check.my-tcp-health-check.self_link}"]
	region                = "${var.region}"

	backend {
		group = "${google_compute_instance_group.us-ig1.self_link}"
	}

	backend {
		group = "${google_compute_instance_group.us-ig2.self_link}"
	}
}

resource "google_compute_forwarding_rule" "my-int-lb-forwarding-rule" {
	name                  = "my-int-lb-forwarding-rule"
	load_balancing_scheme = "INTERNAL"
	ports                 = ["80"]
	network               = "${google_compute_network.my-custom-network.self_link}"
	subnetwork            = "${google_compute_subnetwork.my-custom-subnet.self_link}"
	backend_service       = "${google_compute_region_backend_service.my-int-lb.self_link}"
}

resource "google_compute_firewall" "allow-internal-lb" {
	name    = "allow-internal-lb"
	network = "${google_compute_network.my-custom-network.name}"

	allow {
		protocol = "tcp"
		ports    = ["80", "443"]
	}

	source_ranges = ["10.128.0.0/20"]
	target_tags = ["int-lb"]
}

resource "google_compute_firewall" "allow-health-check" {
	name    = "allow-health-check"
	network = "${google_compute_network.my-custom-network.name}"

	allow {
		protocol = "tcp"
	}

	source_ranges = ["130.211.0.0/22","35.191.0.0/16"]
	target_tags = ["int-lb"]
}

resource "google_compute_instance" "standalone-instance-1" {
	name         = "standalone-instance-1"
	machine_type = "n1-standard-1"
	zone         = "${var.region_zone}"

	tags = ["standalone"]

	disk {
		image = "debian-cloud/debian-8"
	}

	network_interface {
		subnetwork = "${google_compute_subnetwork.my-custom-subnet.name}"
		access_config {
			// Ephemeral IP
		}
	}
}

resource "google_compute_firewall" "allow-ssh-to-standalone" {
	name    = "allow-ssh-to-standalone"
	network = "${google_compute_network.my-custom-network.name}"

	allow {
		protocol = "tcp"
		ports    = ["22"]
	}

	target_tags = ["standalone"]
}
