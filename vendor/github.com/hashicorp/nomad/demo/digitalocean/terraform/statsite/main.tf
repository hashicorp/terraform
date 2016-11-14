variable "size" { default = "1gb" }
variable "region" {}
variable "ssh_keys" {}

resource "atlas_artifact" "statsite-digitalocean" {
  name    = "hashicorp/nomad-demo-statsite"
  type    = "digitalocean.image"
  version = "latest"
}

resource "digitalocean_droplet" "statsite" {
  image    = "${atlas_artifact.statsite-digitalocean.id}"
  name     = "nomad-statsite-${var.region}-${count.index}"
  count    = 1
  size     = "${var.size}"
  region   = "${var.region}"
  ssh_keys = ["${split(",", var.ssh_keys)}"]

  provisioner "remote-exec" {
    inline = "sudo start statsite || true"
  }
}

output "addr" {
  value = "${digitalocean_droplet.statsite.ipv4_address}:8125"
}
