variable "count" {}
variable "image" {}
variable "region" {}
variable "size" { default = "1gb" }
variable "servers" {}
variable "ssh_keys" {}

resource "template_file" "client_config" {
  filename = "${path.module}/client.hcl.tpl"
  vars {
    datacenter = "${var.region}"
    servers    = "${split(",", var.servers)}"
  }
}

resource "digitalocean_droplet" "client" {
  image    = "${var.image}"
  name     = "nomad-client-${var.region}-${count.index}"
  count    = "${var.count}"
  size     = "${var.size}"
  region   = "${var.region}"
  ssh_keys = ["${split(",", var.ssh_keys)}"]

  provisioner "remote-exec" {
    inline = <<CMD
cat > /usr/local/etc/nomad/client.hcl <<EOF
${template_file.client_config.rendered}
EOF
CMD
  }

  provisioner "remote-exec" {
    inline = "sudo start nomad || sudo restart nomad"
  }
}
