---
layout: "aws"
page_title: "AWS: aws_instance"
sidebar_current: "docs-aws-resource-instance"
description: |-
  Provides an EC2 instance resource. This allows instances to be created, updated, and deleted. Instances also support provisioning.
---

# aws\_instance

Provides an EC2 instance resource. This allows instances to be created, updated,
and deleted. Instances also support [provisioning](/docs/provisioners/index.html).

## Example Usage

```
# Create a new instance of the ami-1234 on an m1.small node with an AWS Tag naming it "HelloWorld"
resource "aws_instance" "web" {
    ami = "ami-1234"
    instance_type = "m1.small"
    tags {
        Name = "HelloWorld"
    }
}
```

## Argument Reference

The following arguments are supported:

* `ami` - (Required) The AMI to use for the instance.
* `availability_zone` - (Optional) The AZ to start the instance in.
* `ebs_optimized` - (Optional) If true, the launched EC2 instance will be
     EBS-optimized.
* `instance_type` - (Required) The type of instance to start
* `key_name` - (Optional) The key name to use for the instance.
* `security_groups` - (Optional) A list of security group IDs or names to associate with.
   If you are within a VPC, you'll need to use the security group ID. Otherwise,
   for EC2, use the security group name.
* `subnet_id` - (Optional) The VPC Subnet ID to launch in.
* `associate_public_ip_address` - (Optional) Associate a public ip address with an instance in a VPC.
* `private_ip` - (Optional) Private IP address to associate with the
     instance in a VPC.
* `source_dest_check` - (Optional) Controls if traffic is routed to the instance when
  the destination address does not match the instance. Used for NAT or VPNs. Defaults true.
* `user_data` - (Optional) The user data to provide when launching the instance.
* `iam_instance_profile` - (Optional) The IAM Instance Profile to
  launch the instance with.
* `tags` - (Optional) A mapping of tags to assign to the resource.
* `volume` - (Optional) A list of volumes to add. Their keys are documented below.

Each volume supports the following:

* `size` - (Required) The size of the device in gbs.
* `device_name` - (Required) The name of the device.  Example: "/dev/sda1".
* `volume_type` - (Optional) The type of volume to add.
* `encrypted` - (Optional) If the volume is to be encrypted or not.
* `iops` - (Optional) How many I/O operations per second for volume.  100-4,000.

## Attributes Reference

The following attributes are exported:

* `id` - The instance ID.
* `availability_zone` - The availability zone of the instance.
* `key_name` - The key name of the instance
* `private_dns` - The Private DNS name of the instance
* `private_ip` - The private IP address.
* `public_dns` - The public DNS name of the instance
* `public_ip` - The public IP address.
* `security_groups` - The associated security groups.
* `subnet_id` - The VPC subnet ID.
