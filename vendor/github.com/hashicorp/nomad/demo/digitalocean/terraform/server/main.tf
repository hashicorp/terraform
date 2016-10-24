variable "image" {}
variable "region" {}
variable "size" { default = "8gb" }
variable "ssh_keys" {}
variable "statsite" {}

resource "digitalocean_droplet" "server" {
  image    = "${var.image}"
  name     = "nomad-server-${var.region}-${count.index}"
  count    = 3
  size     = "${var.size}"
  region   = "${var.region}"
  ssh_keys = ["${split(",", var.ssh_keys)}"]

  provisioner "remote-exec" {
    inline = <<CMD
cat > /usr/local/etc/nomad/server.hcl <<EOF
datacenter = "${var.region}"
server {
    enabled = true
    bootstrap_expect = 3
}
advertise {
    rpc = "${self.ipv4_address}:4647"
    serf = "${self.ipv4_address}:4648"
}
telemetry {
    statsite_address = "${var.statsite}"
    disable_hostname = true
}
EOF
CMD
  }

  provisioner "remote-exec" {
    inline = "sudo start nomad || sudo restart nomad"
  }
}

resource "null_resource" "server_join" {
  provisioner "local-exec" {
    command = <<CMD
join() {
  curl -X PUT ${digitalocean_droplet.server.0.ipv4_address}:4646/v1/agent/join?address=$1
}
join ${digitalocean_droplet.server.1.ipv4_address}
join ${digitalocean_droplet.server.2.ipv4_address}
CMD
  }
}

output "addrs" {
  value = "${join(",", digitalocean_droplet.server.*.ipv4_address)}"
}
