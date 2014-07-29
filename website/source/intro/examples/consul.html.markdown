---
layout: "intro"
page_title: "Consul Example"
sidebar_current: "examples-consul"
---

# Consul Example

[Consul](http://www.consul.io) is a tool for service discovery, configuration
and orchestration. The Key/Value store it provides is often used to store
application configuration and information about the infrastructure necessary
to process requests.

Terraform provides a [Consul provider](/docs/providers/consul/index.html) which
can be used to interface with Consul from inside a Terraform configuration.

For our example, we use the [Consul demo cluster](http://demo.consul.io)
to both read configuration and store information about a newly created EC2 instance.
The size of the EC2 instance will be determined by the "tf\_test/size" key in Consul,
and will default to "m1.small" if that key does not exist. Once the instance is created
the "tf\_test/id" and "tf\_test/public\_dns" keys will be set with the computed
values for the instance.

Before we run the example, use the [Web UI](http://demo.consul.io/ui/#/nyc1/kv/)
to set the "tf\_test/size" key to "t1.micro". Once that is done,
copy the configuration into a configuration file ("consul.tf" works fine).
Either provide the AWS credentials as a default value in the configuration
or invoke `apply` with the appropriate variables set.

Once the `apply` has completed, we can see the keys in Consul by
visiting the [Web UI](http://demo.consul.io/ui/#/nyc1/kv/). We can see
that the "tf\_test/id" and "tf\_test/public\_dns" values have been
set.

We can now teardown the infrastructure following the
[instructions here](/intro/getting-started/destroy.html). Because
we set the 'delete' property of two of the Consul keys, Terraform
will cleanup those keys on destroy. We can verify this by using
the Web UI.

The point of this example is to show that Consul can be used with
Terraform both to enable dynamic inputs, but to also store outputs.

Inputs like AMI name, security groups, puppet roles, bootstrap scripts,
etc can all be loaded from Consul. This allows the specifics of an
infrastructure to be decoupled from its overall architecture. This enables
details to be changed without updating the Terraform configuration.

Outputs from Terraform can also be easily stored in Consul. One powerful
features this enables is using Consul for inventory management. If an
application relies on ELB for routing, Terraform can update the application's
configuration directly by setting the ELB address into Consul. Any resource
attribute can be stored in Consul, allowing an operator to capture anything
useful.


## Command

```
terraform apply \
    -var 'aws_access_key=YOUR_KEY' \
    -var 'aws_secret_key=YOUR_KEY'
```

## Configuration

```
# Declare our variables, require access and secret keys
variable "aws_access_key" {}
variable "aws_secret_key" {}
variable "aws_region" {
    default = "us-east-1"
}

# AMI's from http://cloud-images.ubuntu.com/locator/ec2/
variable "aws_amis" {
    default = {
        "eu-west-1": "ami-b1cf19c6",
        "us-east-1": "ami-de7ab6b6",
        "us-west-1": "ami-3f75767a",
        "us-west-2": "ami-21f78e11",
    }
}

# Setup the Consul provisioner to use the demo cluster
provider "consul" {
    address = "demo.consul.io:80"
    datacenter = "nyc1"
}

# Setup an AWS provider
provider "aws" {
    access_key = "${var.aws_access_key}"
    secret_key = "${var.aws_secret_key}"
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
```
