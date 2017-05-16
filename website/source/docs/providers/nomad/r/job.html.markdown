---
layout: "nomad"
page_title: "Nomad: nomad_job"
sidebar_current: "docs-nomad-resource-job"
description: |-
  Manages a job registered in Nomad.
---

# nomad_job

Manages a job registered in Nomad.

This can be used to initialize your cluster with system jobs, common services,
and more. In day to day Nomad use it is common for developers to submit
jobs to Nomad directly, such as for general app deployment. In addition to
these apps, a Nomad cluster often runs core system services that are ideally
setup during infrastructure creation. This resource is ideal for the latter
type of job, but can be used to manage any job within Nomad.

## Example Usage

Registering a job from a jobspec file:

```hcl
resource "nomad_job" "app" {
  jobspec = "${file("${path.module}/job.hcl")}"
}
```

Registering a job from an inline jobspec. This is less realistic but
is an example of how it is possible. More likely, the contents will
be paired with something such as the
[template_file](https://www.terraform.io/docs/providers/template/d/file.html)
resource to render parameterized jobspecs.

```hcl
resource "nomad_job" "app" {
  jobspec = <<EOT
job "foo" {
  datacenters = ["dc1"]
  type = "service"
  group "foo" {
    task "foo" {
      driver = "raw_exec"
      config {
        command = "/bin/sleep"
        args = ["1"]
      }

      resources {
        cpu = 20
        memory = 10
      }

      logs {
        max_files = 3
        max_file_size = 10
      }
    }
  }
}
EOT
}
```

## Argument Reference

The following arguments are supported:

* `jobspec` - (Required) The contents of the jobspec to register.

* `deregister_on_destroy` - (Optional) If true, the job will be deregistered
  when this resource is destroyed in Terraform. Defaults to true.

* `deregister_on_id_change` - (Optional) If true, the job will be deregistered
  if the ID of the job in the jobspec changes. Defaults to true.
