---
layout: "enterprise"
page_title: "Creating AMIs - Packer Artifacts - Terraform Enterprise"
sidebar_current: "docs-enterprise-packerartifacts-amis"
description: |-
  Creating AMI artifacts with Terraform Enterprise.
---

# Creating AMI Artifacts with Terraform Enterprise

In an immutable infrastructure workflow, it's important to version and store
full images (artifacts) to be deployed. This section covers storing [AWS
AMI](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/AMIs.html) images in
Terraform Enterprise to be queried and used later.

Note the actual AMI does _not get stored_. Terraform Enterprise simply keeps the
AMI ID as a reference to the target image. Tools like Terraform can then use
this in a deploy.

### Steps

If you run Packer in Terraform Enterprise, the following will happen after a [push](/docs/enterprise/packer/builds/starting.html):

1. Terraform Enterprise will run `packer build` against your template in our
infrastructure. This spins up an AWS instance in your account and provisions it
with any specified provisioners

2. Packer stops the instance and stores the result as an AMI in AWS under your
account. This then returns an ID (the artifact) that it passes to the
post-processor

3. The post-processor creates and uploads the new artifact version with the ID
in Terraform Enterprise of the type `amazon.image` for use later

### Example

Below is a complete example Packer template that starts an AWS instance.

```json
{
  "push": {
    "name": "my-username/frontend"
  },
  "provisioners": [],
  "builders": [
    {
      "type": "amazon-ebs",
      "access_key": "",
      "secret_key": "",
      "region": "us-east-1",
      "source_ami": "ami-2ccc7a44",
      "instance_type": "c3.large",
      "ssh_username": "ubuntu",
      "ami_name": "Terraform Enterprise Example {{ timestamp }}"
    }
  ],
  "post-processors": [
    {
      "type": "atlas",
      "artifact": "my-username/web-server",
      "artifact_type": "amazon.image"
    }
  ]
}
```
