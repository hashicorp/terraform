---
layout: "enterprise"
page_title: "Monolithic Artifacts - FAQ - Terraform Enterprise"
sidebar_current: "docs-enterprise-faq-monolithic"
description: |-
  How do I build multiple applications into one artifact?
---

# Monolithic Artifacts

*How do I build multiple applications into one artifact?*

Create your new Applications in Terraform Enterprise using the application
compilation feature.

You can either link each Application to the single Build Template you will be
using to create the monolithic artifact, or run periodic Packer builds.

Each time an Application is pushed, it will store the new application version in
the artifact registry as a tarball. These will be available for you to download
at build-time on the machines they belong.

Here's an example `compile.json` template that you will include with the rest of
your application files that do the compiling:


```json
{
  "variables": {
    "app_slug": "{{ env `ATLAS_APPLICATION_SLUG` }}"
  },
  "builders": [
    {
      "type": "docker",
      "image": "ubuntu:14.04",
      "commit": true
    }
  ],
  "provisioners": [
    {
      "type": "shell",
      "inline": [
        "apt-get -y update"
      ]
    },
    {
      "type": "file",
      "source": ".",
      "destination": "/tmp/app"
    },
    {
      "type": "shell",
      "inline": [
        "cd /tmp/app",
        "make"
      ]
    },
    {
      "type": "file",
      "source": "/tmp/compiled-app.tar.gz",
      "destination": "compiled-app.tar.gz",
      "direction": "download"
    }
  ],
  "post-processors": [
    [
      {
        "type": "artifice",
        "files": ["compiled-app.tar.gz"]
      },
      {
        "type": "atlas",
        "artifact": "{{user `app_slug` }}",
        "artifact_type": "archive"
      }
    ]
  ]
}
```

In your Packer template, you can download each of the latest applications
artifacts onto the host using the shell provisioner:


```text
$ curl -L -H "X-Atlas-Token: ${ATLAS_TOKEN}" https://atlas.hashicorp.com/api/v1/artifacts/hashicorp/example/archive/latest/file -o example.tar.gz
```

Here's an example Packer template:

```json
{
  "variables": {
    "atlas_username": "{{env `ATLAS_USERNAME`}}",
    "aws_access_key": "{{env `AWS_ACCESS_KEY_ID`}}",
    "aws_secret_key": "{{env `AWS_SECRET_ACCESS_KEY`}}",
    "aws_region":     "{{env `AWS_DEFAULT_REGION`}}",
    "instance_type":  "c3.large",
    "source_ami":     "ami-9a562df2",
    "name":           "example",
    "ssh_username":   "ubuntu",
    "app_dir":        "/app"
  },
  "push": {
    "name": "{{user `atlas_username`}}/{{user `name`}}",
    "vcs": false
  },
  "builders": [
    {
      "type":            "amazon-ebs",
      "access_key":      "{{user `aws_access_key`}}",
      "secret_key":      "{{user `aws_secret_key`}}",
      "region":          "{{user `aws_region`}}",
      "vpc_id":          "",
      "subnet_id":       "",
      "instance_type":   "{{user `instance_type`}}",
      "source_ami":      "{{user `source_ami`}}",
      "ami_regions":     [],
      "ami_name":        "{{user `name`}} {{timestamp}}",
      "ami_description": "{{user `name`}} AMI",
      "run_tags":        { "ami-create": "{{user `name`}}" },
      "tags":            { "ami": "{{user `name`}}" },
      "ssh_username":    "{{user `ssh_username`}}",
      "ssh_timeout":     "10m",
      "ssh_private_ip":  false,
      "associate_public_ip_address": true
    }
  ],
  "provisioners": [
    {
      "type": "shell",
      "execute_command": "echo {{user `ssh_username`}} | {{ .Vars }} sudo -E -S sh '{{ .Path }}'",
      "inline": [
        "apt-get -y update",
        "apt-get -y upgrade",
        "apt-get -y install curl unzip tar",
        "mkdir -p {{user `app_dir`}}",
        "chmod a+w {{user `app_dir`}}",
        "cd /tmp",
        "curl -L -H 'X-Atlas-Token: ${ATLAS_TOKEN}' https://atlas.hashicorp.com/api/v1/artifacts/{{user `atlas_username`}}/{{user `name`}}/archive/latest/file -o example.tar.gz",
        "tar -xzf example.tar.gz -C {{user `app_dir`}}"
      ]
    }
  ],
  "post-processors": [
    {
      "type": "atlas",
      "artifact": "{{user `atlas_username`}}/{{user `name`}}",
      "artifact_type": "amazon.image",
      "metadata": {
        "created_at": "{{timestamp}}"
      }
    }
  ]
}
```

Once downloaded, you can place each application slug where it needs to go to
produce the monolithic artifact your are accustom to.
