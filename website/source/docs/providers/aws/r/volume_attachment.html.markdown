---
layout: "aws"
page_title: "AWS: aws_volume_attachment"
sidebar_current: "docs-aws-resource-volume-attachment"
description: |-
  Provides an AWS EBS Volume Attachment
---

# aws\_volume\_attachment

Provides an AWS EBS Volume Attachment as a top level resource, to attach and
detach volumes from AWS Instances.

~> **NOTE on EBS block devices:** If you use `ebs_block_device` on an `aws_instance`, Terraform will assume management over the full set of non-root EBS block devices for the instance, and treats additional block devices as drift. For this reason, `ebs_block_device` cannot be mixed with external `aws_ebs_volume` + `aws_ebs_volume_attachment` resources for a given instance.

## Example Usage

```
resource "aws_volume_attachment" "ebs_att" {
  device_name = "/dev/sdh"
  volume_id = "${aws_ebs_volume.example.id}"
  instance_id = "${aws_instance.web.id}"
}

resource "aws_instance" "web" {
  ami = "ami-21f78e11"
  availability_zone = "us-west-2a"
  instance_type = "t1.micro"
  tags {
    Name = "HelloWorld"
  }
}

resource "aws_ebs_volume" "example" {
  availability_zone = "us-west-2a"
  size = 1
}
```

## Argument Reference

The following arguments are supported:

* `device_name` - (Required) The device name to expose to the instance (for 
example, `/dev/sdh` or `xvdh`)
* `instance_id` - (Required) ID of the Instance to attach to
* `volume_id` - (Required) ID of the Volume to be attached
* `force_detach` - (Optional, Boolean) Set to `true` if you want to force the
volume to detach. Useful if previous attempts failed, but use this option only 
as a last resort, as this can result in **data loss**. See 
[Detaching an Amazon EBS Volume from an Instance][1] for more information.

## Attributes Reference

* `device_name` - The device name exposed to the instance
* `instance_id` - ID of the Instance
* `volume_id` - ID of the Volume 

[1]: https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ebs-detaching-volume.html
