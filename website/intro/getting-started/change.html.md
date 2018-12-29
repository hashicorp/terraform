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

Let's modify the `ami` of our instance. Edit the `aws_instance.example`
resource in your configuration and change it to the following:

```hcl
resource "aws_instance" "example" {
  ami           = "ami-b374d5a5"
  instance_type = "t2.micro"
}
```

~> **Note:** EC2 Classic users please use AMI `ami-656be372` and type `t1.micro`

We've changed the AMI from being an Ubuntu 16.04 LTS AMI to being
an Ubuntu 16.10 AMI. Terraform configurations are meant to be
changed like this. You can also completely remove resources
and Terraform will know to destroy the old one.

## Apply Changes

After changing the configuration, run `terraform apply` again to see how
Terraform will apply this change to the existing resources.

```
$ terraform apply
# ...

-/+ aws_instance.example
    ami:                      "ami-2757f631" => "ami-b374d5a5" (forces new resource)
    availability_zone:        "us-east-1a" => "<computed>"
    ebs_block_device.#:       "0" => "<computed>"
    ephemeral_block_device.#: "0" => "<computed>"
    instance_state:           "running" => "<computed>"
    instance_type:            "t2.micro" => "t2.micro"
    private_dns:              "ip-172-31-17-94.ec2.internal" => "<computed>"
    private_ip:               "172.31.17.94" => "<computed>"
    public_dns:               "ec2-54-82-183-4.compute-1.amazonaws.com" => "<computed>"
    public_ip:                "54.82.183.4" => "<computed>"
    subnet_id:                "subnet-1497024d" => "<computed>"
    vpc_security_group_ids.#: "1" => "<computed>"
```

The prefix `-/+` means that Terraform will destroy and recreate
the resource, rather than updating it in-place. While some attributes
can be updated in-place (which are shown with the `~` prefix), changing the
AMI for an EC2 instance requires recreating it. Terraform handles these details
for you, and the execution plan makes it clear what Terraform will do.

Additionally, the execution plan shows that the AMI change is what
required the resource to be replaced. Using this information,
you can adjust your changes to possibly avoid destroy/create updates
if they are not acceptable in some situations.

Once again, Terraform prompts for approval of the execution plan before
proceeding. Answer `yes` to execute the planned steps:


```
# ...
aws_instance.example: Refreshing state... (ID: i-64c268fe)
aws_instance.example: Destroying...
aws_instance.example: Destruction complete
aws_instance.example: Creating...
  ami:                      "" => "ami-b374d5a5"
  availability_zone:        "" => "<computed>"
  ebs_block_device.#:       "" => "<computed>"
  ephemeral_block_device.#: "" => "<computed>"
  instance_state:           "" => "<computed>"
  instance_type:            "" => "t2.micro"
  key_name:                 "" => "<computed>"
  placement_group:          "" => "<computed>"
  private_dns:              "" => "<computed>"
  private_ip:               "" => "<computed>"
  public_dns:               "" => "<computed>"
  public_ip:                "" => "<computed>"
  root_block_device.#:      "" => "<computed>"
  security_groups.#:        "" => "<computed>"
  source_dest_check:        "" => "true"
  subnet_id:                "" => "<computed>"
  tenancy:                  "" => "<computed>"
  vpc_security_group_ids.#: "" => "<computed>"
aws_instance.example: Still creating... (10s elapsed)
aws_instance.example: Still creating... (20s elapsed)
aws_instance.example: Creation complete

Apply complete! Resources: 1 added, 0 changed, 1 destroyed.

# ...
```

As indicated by the execution plan, Terraform first destroyed the existing
instance and then created a new one in its place. You can use `terraform show`
again to see the new values associated with this instance.

## Next

You've now seen how easy it is to modify infrastructure with
Terraform. Feel free to play around with this more before continuing.
In the next section we're going to [destroy our infrastructure](/intro/getting-started/destroy.html).
