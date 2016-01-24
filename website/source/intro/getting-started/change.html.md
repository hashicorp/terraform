---
layout: "intro"
page_title: "Change Infrastructure"
sidebar_current: "gettingstarted-change"
description: |-
  In the previous page, you created your first infrastructure with Terraform: a single EC2 instance. In this page, we're going to modify that resource, and see how Terraform handles change.
---

# Change Infrastructure

In the previous page, you created your first infrastructure with
Terraform: a single EC2 instance. In this page, we're going to
modify that resource, and see how Terraform handles change.

Infrastructure is continuously evolving, and Terraform was built
to help manage and enact that change. As you change Terraform
configurations, Terraform builds an execution plan that only
modifies what is necessary to reach your desired state.

By using Terraform to change infrastructure, you can version
control not only your configurations but also your state so you
can see how the infrastructure evolved over time.

## Configuration

Let's modify the `ami` of our instance. Edit the "aws\_instance.example"
resource in your configuration and change it to the following:

```
resource "aws_instance" "example" {
	ami = "ami-b8b061d0"
	instance_type = "t1.micro"
}
```

We've changed the AMI from being an Ubuntu 14.04 AMI to being
an Ubuntu 12.04 AMI. Terraform configurations are meant to be
changed like this. You can also completely remove resources
and Terraform will know to destroy the old one.

## Execution Plan

Let's see what Terraform will do with the change we made.

```
$ terraform plan
...

-/+ aws_instance.example
    ami:               "ami-408c7f28" => "ami-b8b061d0" (forces new resource)
    availability_zone: "us-east-1c" => "<computed>"
    key_name:          "" => "<computed>"
    private_dns:       "domU-12-31-39-12-38-AB.compute-1.internal" => "<computed>"
    private_ip:        "10.200.59.89" => "<computed>"
    public_dns:        "ec2-54-81-21-192.compute-1.amazonaws.com" => "<computed>"
    public_ip:         "54.81.21.192" => "<computed>"
    security_groups:   "" => "<computed>"
    subnet_id:         "" => "<computed>"
```

The prefix "-/+" means that Terraform will destroy and recreate
the resource, versus purely updating it in-place. While some attributes
can do in-place updates (which are shown with a "~" prefix), AMI
changing on EC2 instance requires a new resource. Terraform handles
these details for you, and the execution plan makes it clear what
Terraform will do.

Additionally, the plan output shows that the AMI change is what
necessitated the creation of a new resource. Using this information,
you can tweak your changes to possibly avoid destroy/create updates
if you didn't want to do them at this time.

## Apply

From the plan, we know what will happen. Let's apply and enact
the change.

```
$ terraform apply
aws_instance.example: Destroying...
aws_instance.example: Modifying...
  ami: "ami-408c7f28" => "ami-b8b061d0"

Apply complete! Resources: 0 added, 1 changed, 1 destroyed.

...
```

As the plan predicted, Terraform started by destroying our old
instance, then creating the new one. You can use `terraform show`
again to see the new properties associated with this instance.

## Next

You've now seen how easy it is to modify infrastructure with
Terraform. Feel free to play around with this more before continuing.
In the next section we're going to [destroy our infrastructure](/intro/getting-started/destroy.html).
