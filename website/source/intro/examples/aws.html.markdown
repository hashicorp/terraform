---
layout: "intro"
page_title: "Basic Two-Tier AWS Architecture"
sidebar_current: "examples-aws"
---

# Basic Two-Tier AWS Architecture

This provides a template for running a simple two-tier architecture on Amazon
Web services.

The basic premise is you have stateless app servers running behind
an ELB serving traffic.

To simplify the example, this intentionally ignores deploying and
getting your application onto the servers. However, you could do so either via
[provisioners](/docs/provisioners/index.html) and a configuration
management tool, or by pre-baking configured AMIs with
[Packer](http://www.packer.io).

After you run `terraform apply` on this configuration, it will
automatically output the DNS address of the ELB. After your instance
registers, this should respond with the default nginx web page.

The configuration file contains comments describing each
resource.

## Command

```
 terraform apply \
    -var 'aws_access_key=YOUR_ACCESS_KEY' \
    -var 'aws_secret_key=YOUR_SECRET_KEY' \
    -var 'key_path=/path/to/key/pair.pem' \
    -var 'key_name=keypair-name'
```

## Configuration

```
variable "aws_access_key" {}
variable "aws_secret_key" {}
variable "key_path" {}
variable "key_name" {}
variable "aws_region" {
    default = "us-west-2"
}

# Ubuntu Precise 12.04 LTS (x64)
variable "aws_amis" {
    default = {
        "eu-west-1": "ami-b1cf19c6",
        "us-east-1": "ami-de7ab6b6",
        "us-west-1": "ami-3f75767a",
        "us-west-2": "ami-21f78e11"
    }
}

# Specify the provider and access details
provider "aws" {
    access_key = "${var.aws_access_key}"
    secret_key = "${var.aws_secret_key}"
    region = "${var.aws_region}"
}

# Our default security group to access
# the instances over SSH and HTTP
resource "aws_security_group" "default" {
    name = "terraform_example"
    description = "Used in the terraform"

    # SSH access from anywhere
    ingress {
        from_port = 22
        to_port = 22
        protocol = "tcp"
        cidr_blocks = ["0.0.0.0/0"]
    }

    # HTTP access from anywhere
    ingress {
        from_port = 80
        to_port = 80
        protocol = "tcp"
        cidr_blocks = ["0.0.0.0/0"]
    }
}


resource "aws_elb" "web" {
  name = "terraform-example-elb"

  # The same availability zone as our instance
  availability_zones = ["${aws_instance.web.availability_zone}"]

  listener {
    instance_port = 80
    instance_protocol = "http"
    lb_port = 80
    lb_protocol = "http"
  }

  # The instance is registered automatically
  instances = ["${aws_instance.web.id}"]
}


resource "aws_instance" "web" {
  # The connection block tells our provisioner how to
  # communicate with the resource (instance)
  connection {
    # The default username for our AMI
    user = "ubuntu"

    # The path to your keyfile
    key_file = "${var.key_path}"
  }

  instance_type = "m1.small"

  # Lookup the correct AMI based on the region
  # we specified
  ami = "${lookup(var.aws_amis, var.aws_region)}"

  # The name of our SSH keypair you've created and downloaded
  # from the AWS console.
  #
  # https://console.aws.amazon.com/ec2/v2/home?region=us-west-2#KeyPairs:
  #
  key_name = "${var.key_name}"

  # Our Security group to allow HTTP and SSH access
  security_groups = ["${aws_security_group.default.name}"]

  # We run a remote provisioner on the instance after creating it.
  # In this case, we just install nginx and start it. By default,
  # this should be on port 80
  provisioner "remote-exec" {
    inline = [
        "sudo apt-get -y update",
        "sudo apt-get -y install nginx",
        "sudo service nginx start",
    ]
  }
}

output "address" {
  value = "${aws_elb.web.dns_name}"
}
```
