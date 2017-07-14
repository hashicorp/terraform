---
layout: "intro"
page_title: "Resource Dependencies"
sidebar_current: "gettingstarted-deps"
description: |-
  In this page, we're going to introduce resource dependencies, where we'll not only see a configuration with multiple resources for the first time, but also scenarios where resource parameters use information from other resources.
---

# Resource Dependencies

In this page, we're going to introduce resource dependencies,
where we'll not only see a configuration with multiple resources
for the first time, but also scenarios where resource parameters
use information from other resources.

Up to this point, our example has only contained a single resource.
Real infrastructure has a diverse set of resources and resource
types. Terraform configurations can contain multiple resources,
multiple resource types, and these types can even span multiple
providers.

On this page, we'll show a basic example of multiple resources
and how to reference the attributes of other resources to configure
subsequent resources.

## Assigning an Elastic IP

We'll improve our configuration by assigning an elastic IP to
the EC2 instance we're managing. Modify your `example.tf` and
add the following:

```hcl
resource "aws_eip" "ip" {
  instance = "${aws_instance.example.id}"
}
```

This should look familiar from the earlier example of adding
an EC2 instance resource, except this time we're building
an "aws\_eip" resource type. This resource type allocates
and associates an
[elastic IP](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/elastic-ip-addresses-eip.html)
to an EC2 instance.

The only parameter for
[aws\_eip](/docs/providers/aws/r/eip.html) is "instance" which
is the EC2 instance to assign the IP to. For this value, we
use an interpolation to use an attribute from the EC2 instance
we managed earlier.

The syntax for this interpolation should be straightforward:
it requests the "id" attribute from the "aws\_instance.example"
resource.

## Plan and Execute

Run `terraform plan` to view the execution plan. The output
will look something like the following:

```
$ terraform plan

+ aws_eip.ip
    allocation_id:     "<computed>"
    association_id:    "<computed>"
    domain:            "<computed>"
    instance:          "${aws_instance.example.id}"
    network_interface: "<computed>"
    private_ip:        "<computed>"
    public_ip:         "<computed>"

+ aws_instance.example
    ami:                      "ami-b374d5a5"
    availability_zone:        "<computed>"
    ebs_block_device.#:       "<computed>"
    ephemeral_block_device.#: "<computed>"
    instance_state:           "<computed>"
    instance_type:            "t2.micro"
    key_name:                 "<computed>"
    placement_group:          "<computed>"
    private_dns:              "<computed>"
    private_ip:               "<computed>"
    public_dns:               "<computed>"
    public_ip:                "<computed>"
    root_block_device.#:      "<computed>"
    security_groups.#:        "<computed>"
    source_dest_check:        "true"
    subnet_id:                "<computed>"
    tenancy:                  "<computed>"
    vpc_security_group_ids.#: "<computed>"
```

Terraform will create two resources: the instance and the elastic
IP. In the "instance" value for the "aws\_eip", you can see the
raw interpolation is still present. This is because this variable
won't be known until the "aws\_instance" is created. It will be
replaced at apply-time.

Next, run `terraform apply`. The output will look similar to the
following:

```
$ terraform apply
aws_instance.example: Creating...
  ami:                      "" => "ami-b374d5a5"
  instance_type:            "" => "t2.micro"
  [..]
aws_instance.example: Still creating... (10s elapsed)
aws_instance.example: Creation complete
aws_eip.ip: Creating...
  allocation_id:     "" => "<computed>"
  association_id:    "" => "<computed>"
  domain:            "" => "<computed>"
  instance:          "" => "i-f3d77d69"
  network_interface: "" => "<computed>"
  private_ip:        "" => "<computed>"
  public_ip:         "" => "<computed>"
aws_eip.ip: Creation complete

Apply complete! Resources: 2 added, 0 changed, 0 destroyed.
```

It is clearer to see from actually running Terraform, but
Terraform creates the EC2 instance before the elastic IP
address. Due to the interpolation earlier where the elastic
IP requires the ID of the EC2 instance, Terraform is able
to infer a dependency, and knows to create the instance
first.

## Implicit and Explicit Dependencies

Most dependencies in Terraform are implicit: Terraform is able
to infer dependencies based on usage of attributes of other
resources.

Using this information, Terraform builds a graph of resources.
This tells Terraform not only in what order to create resources,
but also what resources can be created in parallel. In our example,
since the IP address depended on the EC2 instance, they could
not be created in parallel.

Implicit dependencies work well and are usually all you ever need.
However, you can also specify explicit dependencies with the
`depends_on` parameter which is available on any resource. For example,
we could modify the "aws\_eip" resource to the following, which
effectively does the same thing and is redundant:

```hcl
resource "aws_eip" "ip" {
  instance   = "${aws_instance.example.id}"
  depends_on = ["aws_instance.example"]
}
```

If you're ever unsure about the dependency chain that Terraform
is creating, you can use the [`terraform graph` command](/docs/commands/graph.html) to view
the graph. This command outputs a dot-formatted graph which can be
viewed with
[Graphviz](http://www.graphviz.org/).

## Non-Dependent Resources

We can now augment the configuration with another EC2 instance.
Because this doesn't rely on any other resource, it can be
created in parallel to everything else.

```hcl
resource "aws_instance" "another" {
  ami           = "ami-b374d5a5"
  instance_type = "t2.micro"
}
```

You can view the graph with `terraform graph` to see that
nothing depends on this and that it will likely be created
in parallel.

Before moving on, remove this resource from your configuration
and `terraform apply` again to destroy it. We won't use the
second instance anymore in the getting started guide.

## Next

In this page you were introduced to both multiple resources
as well as basic resource dependencies and resource attribute
interpolation.

Moving on, [we'll use provisioners](/intro/getting-started/provision.html)
to do some basic bootstrapping of our launched instance.
