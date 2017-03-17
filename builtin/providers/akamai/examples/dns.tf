resource "digitalocean_droplet" "my-droplet" {
  image = "ubuntu-16-04-x64"
  name = "terraform-example"
  region = "nyc3"
  size = "512mb"
}

resource "digitalocean_floating_ip" "load-balancer" {
  droplet_id = "${digitalocean_droplet.my-droplet.id}"
  region = "${digitalocean_droplet.my-droplet.region}"
}

resource "akamai_fastdns_record" "origin" {
  hostname = "example.org"
  name = "origin"
  type = "a"
  targets = ["${digitalocean_floating_ip.load-balancer.ip_address}"]
}

resource "akamai_fastdns_zone" "edge" {
  hostname = "example.org"
  name = "www"
  type = "cname"
  target = "origin.example.org"
}