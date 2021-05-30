provider "aws" {}
resource "aws_instance" "foo" {}

provider "do" {}
resource "do_droplet" "bar" {}
