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

## Apply Changes

Run `terraform apply` to see how Terraform plans to apply this change.
The output will look similar to the following:

```
$ terraform apply

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

As usual, Terraform prompts for confirmation before making any changes.
Answer `yes` to apply. The continued output will look similar to the
following:

```
# ...
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

As shown above, Terraform created the EC2 instance before creating the Elastic
IP address. Due to the interpolation expression that passes the ID of the EC2
instance to the Elastic IP address, Terraform is able to infer a dependency,
and knows it must create the instance first.

## Implicit and Explicit Dependencies

By studying the resource attributes used in interpolation expressions,
Terraform can automatically infer when one resource depends on another.
In the example above, the expression `${aws_instance.example.id}` creates
an _implicit dependency_ on the `aws_instance` named `example`.

Terraform uses this dependency information to determine the correct order
in which to create the different resources. In the example above, Terraform
knows that the `aws_instance` must be created before the `aws_eip`.

Implicit dependencies via interpolation expressions are the primary way
to inform Terraform about these relationships, and should be used whenever
possible.

Sometimes there are dependencies between resources that are _not_ visible to
Terraform. The `depends_on` argument is accepted by any resource and accepts
a list of resources to create _explicit dependencies_ for.

For example, perhaps an application we will run on our EC2 instance expects
to use a specific Amazon S3 bucket, but that dependency is configured
inside the application code and thus not visible to Terraform. In
that case, we can use `depends_on` to explicitly declare the dependency:

```hcl
# New resource for the S3 bucket our application will use.
resource "aws_s3_bucket" "example" {
  # NOTE: S3 bucket names must be unique across _all_ AWS accounts, so
  # this name must be changed before applying this example to avoid naming
  # conflicts.
  bucket = "terraform-getting-started-guide"
  acl    = "private"
}

# Change the aws_instance we declared earlier to now include "depends_on"
resource "aws_instance" "example" {
  ami           = "ami-2757f631"
  instance_type = "t2.micro"

  # Tells Terraform that this EC2 instance must be created only after the
  # S3 bucket has been created.
  depends_on = ["aws_s3_bucket.example"]
}
```

## Non-Dependent Resources

We can continue to build this configuration by adding another EC2 instance:

```hcl
resource "aws_instance" "another" {
  ami           = "ami-b374d5a5"
  instance_type = "t2.micro"
}
```

Because this new instance does not depend on any other resource, it can
be created in parallel with the other resources. Where possible, Terraform
will perform operations concurrently to reduce the total time taken to
apply changes.

Before moving on, remove this new resource from your configuration and
run `terraform apply` again to destroy it. We won't use this second instance
any further in the getting started guide.

## Next

In this page you were introduced to using multiple resources, interpolating
attributes from one resource into another, and declaring dependencies between
resources to define operation ordering.

In the next section, [we'll use provisioners](/intro/getting-started/provision.html)
to do some basic bootstrapping of our launched instance.
