# An example of how to connect two GCE networks with a VPN
provider "google" {
    account_file = "${file("~/gce/account.json")}"
    project = "${var.project}"
    region = "${var.region1}"
}

# Create the two networks we want to join. They must have seperate, internal
# ranges.
resource "google_compute_network" "network1" {
    name = "network1"
    ipv4_range = "10.120.0.0/16"
}

resource "google_compute_network" "network2" {
    name = "network2"
    ipv4_range = "10.121.0.0/16"
}

# Attach a VPN gateway to each network.
resource "google_compute_vpn_gateway" "target_gateway1" {
    name = "vpn1"
    network = "${google_compute_network.network1.self_link}"
    region = "${var.region1}"
}

resource "google_compute_vpn_gateway" "target_gateway2" {
    name = "vpn2"
    network = "${google_compute_network.network2.self_link}"
    region = "${var.region2}"
}

# Create an outward facing static IP for each VPN that will be used by the
# other VPN to connect.
resource "google_compute_address" "vpn_static_ip1" {
    name = "vpn-static-ip1"
    region = "${var.region1}"
}

resource "google_compute_address" "vpn_static_ip2" {
    name = "vpn-static-ip2"
    region = "${var.region2}"
}

# Forward IPSec traffic coming into our static IP to our VPN gateway.
resource "google_compute_forwarding_rule" "fr1_esp" {
    name = "fr1-esp"
    region = "${var.region1}"
    ip_protocol = "ESP"
    ip_address = "${google_compute_address.vpn_static_ip1.address}"
    target = "${google_compute_vpn_gateway.target_gateway1.self_link}"
}

resource "google_compute_forwarding_rule" "fr2_esp" {
    name = "fr2-esp"
    region = "${var.region2}"
    ip_protocol = "ESP"
    ip_address = "${google_compute_address.vpn_static_ip2.address}"
    target = "${google_compute_vpn_gateway.target_gateway2.self_link}"
}

# The following two sets of forwarding rules are used as a part of the IPSec
# protocol
resource "google_compute_forwarding_rule" "fr1_udp500" {
    name = "fr1-udp500"
    region = "${var.region1}"
    ip_protocol = "UDP"
    port_range = "500"
    ip_address = "${google_compute_address.vpn_static_ip1.address}"
    target = "${google_compute_vpn_gateway.target_gateway1.self_link}"
}

resource "google_compute_forwarding_rule" "fr2_udp500" {
    name = "fr2-udp500"
    region = "${var.region2}"
    ip_protocol = "UDP"
    port_range = "500"
    ip_address = "${google_compute_address.vpn_static_ip2.address}"
    target = "${google_compute_vpn_gateway.target_gateway2.self_link}"
}

resource "google_compute_forwarding_rule" "fr1_udp4500" {
    name = "fr1-udp4500"
    region = "${var.region1}"
    ip_protocol = "UDP"
    port_range = "4500"
    ip_address = "${google_compute_address.vpn_static_ip1.address}"
    target = "${google_compute_vpn_gateway.target_gateway1.self_link}"
}

resource "google_compute_forwarding_rule" "fr2_udp4500" {
    name = "fr2-udp4500"
    region = "${var.region2}"
    ip_protocol = "UDP"
    port_range = "4500"
    ip_address = "${google_compute_address.vpn_static_ip2.address}"
    target = "${google_compute_vpn_gateway.target_gateway2.self_link}"
}

# Each tunnel is responsible for encrypting and decrypting traffic exiting
# and leaving its associated gateway
resource "google_compute_vpn_tunnel" "tunnel1" {
    name = "tunnel1"
    region = "${var.region1}"
    peer_ip = "${google_compute_address.vpn_static_ip2.address}"
    shared_secret = "a secret message"
    target_vpn_gateway = "${google_compute_vpn_gateway.target_gateway1.self_link}"
    depends_on = ["google_compute_forwarding_rule.fr1_udp500",
        "google_compute_forwarding_rule.fr1_udp4500",
        "google_compute_forwarding_rule.fr1_esp"]
}

resource "google_compute_vpn_tunnel" "tunnel2" {
    name = "tunnel2"
    region = "${var.region2}"
    peer_ip = "${google_compute_address.vpn_static_ip1.address}"
    shared_secret = "a secret message"
    target_vpn_gateway = "${google_compute_vpn_gateway.target_gateway2.self_link}"
    depends_on = ["google_compute_forwarding_rule.fr2_udp500",
        "google_compute_forwarding_rule.fr2_udp4500",
        "google_compute_forwarding_rule.fr2_esp"]
}

# Each route tells the associated network to send all traffic in the dest_range
# through the VPN tunnel
resource "google_compute_route" "route1" {
    name = "route1"
    network = "${google_compute_network.network1.name}"
    next_hop_vpn_tunnel = "${google_compute_vpn_tunnel.tunnel1.self_link}"
    dest_range = "${google_compute_network.network2.ipv4_range}"
    priority = 1000
}

resource "google_compute_route" "route2" {
    name = "route2"
    network = "${google_compute_network.network2.name}"
    next_hop_vpn_tunnel = "${google_compute_vpn_tunnel.tunnel2.self_link}"
    dest_range = "${google_compute_network.network1.ipv4_range}"
    priority = 1000
}

# We want to allow the two networks to communicate, so we need to unblock
# them in the firewall
resource "google_compute_firewall" "network1-allow-network1" {
    name = "network1-allow-network1"
    network = "${google_compute_network.network1.name}"
    source_ranges = ["${google_compute_network.network1.ipv4_range}"]
    allow {
        protocol = "tcp"
    }
    allow {
        protocol = "udp"
    }
    allow {
        protocol = "icmp"
    }
}

resource "google_compute_firewall" "network1-allow-network2" {
    name = "network1-allow-network2"
    network = "${google_compute_network.network1.name}"
    source_ranges = ["${google_compute_network.network2.ipv4_range}"]
    allow {
        protocol = "tcp"
    }
    allow {
        protocol = "udp"
    }
    allow {
        protocol = "icmp"
    }
}
