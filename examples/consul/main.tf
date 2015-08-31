# Setup the Consul provisioner to use the demo cluster
provider "consul" {
  address = "demo.consul.io:80"
  datacenter = "nyc1"
}

# Setup an AWS provider
provider "aws" {
  region = "${var.aws_region}"
}

# Setup a key in Consul to provide inputs
resource "consul_keys" "input" {
  key {
    name = "size"
    path = "tf_test/size"
    default = "m1.small"
  }
}

# Setup a new AWS instance using a dynamic ami and
# instance type
resource "aws_instance" "test" {
  ami = "${lookup(var.aws_amis, var.aws_region)}"
  instance_type = "${consul_keys.input.var.size}"
}

# Setup a key in Consul to store the instance id and
# the DNS name of the instance
resource "consul_keys" "test" {
  key {
    name = "id"
    path = "tf_test/id"
    value = "${aws_instance.test.id}"
    delete = true
  }
  key {
    name = "address"
    path = "tf_test/public_dns"
    value = "${aws_instance.test.public_dns}"
    delete = true
  }
}
