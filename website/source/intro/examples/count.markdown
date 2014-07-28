---
layout: "intro"
page_title: "Count"
sidebar_current: "examples-count"
---

# Count Example

The count parameter on resources can simplify configurations
and let you scale resources by simply incrementing a number.

Additionally, variables can be used to expand a list of resources
for use elsewhere.

## Command

```
 terraform apply \
    -var 'aws_access_key=YOUR_ACCESS_KEY' \
    -var 'aws_secret_key=YOUR_SECRET_KEY'
```

## Configuration

```
variable "aws_access_key" {}
variable "aws_secret_key" {}
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

resource "aws_elb" "web" {
  name = "terraform-example-elb"

  # The same availability zone as our instances
  availability_zones = ["${aws_instance.web.*.availability_zone}"]

  listener {
    instance_port = 80
    instance_protocol = "http"
    lb_port = 80
    lb_protocol = "http"
  }

  # The instances are registered automatically
  instances = ["${aws_instance.web.*.id}"]
}


resource "aws_instance" "web" {
  instance_type = "m1.small"
  ami = "${lookup(var.aws_amis, var.aws_region)}"

  # This will create 4 instances
  count = 4
}

output "address" {
  value = "Instances: ${aws_instance.web.*.id}"
}
```
