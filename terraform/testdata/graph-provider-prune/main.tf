provider "aws" {}
provider "digitalocean" {}
provider "openstack" {}

resource "aws_load_balancer" "weblb" {}
