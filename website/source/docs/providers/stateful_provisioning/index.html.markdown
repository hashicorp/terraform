---
layout: "stateful_provisioning"
page_title: "Provider: Stateful Provisioning"
sidebar_current: "docs-stateful-provisioning-index"
description: |-
  The stateful provisioning provider allows externally-orchestrated deployment via arbitrary provisioners.
---

# Stateful Provisioning

The usual and recommended mode for Terraform deployment is to have Terraform
manage particular resources: creating, updating and deleting them to converge
on the configuration given.

Some applications do not lend themselves well to this deployment model and are
instead deployed by executing a sequence of provisioning steps against
already-existing resources.

The Stateful Provisioning resource allows Terraform to be used to orchestrate
such a deployment, by representing the deployment itself as a resource. Since
Terraform cannot directly discover and manage the application state, the user
must instead help Terraform understand the intended state by representing it as
a string called the *state key*. For many applications a reasonable state key
would be the version control revision id that is being deployed, or the
unique name or location of a deployment artifact.

Each time a stateful provisioning resource is applied, Terraform will check
to see if the state key has changed since the previous run, and if it *has*
then the resource's associated provisioners will be re-run, allowing arbitrary
actions to be taken each time the state key changes. No action will be taken
if the state key is unchanged.

The stateful provisioning resource is intended to be used with one or more
provisioners, defining the actions to be taken each time the state key
changes.

In order to represent the changes within Terraform's resource diffing model,
each new run of the provisioners is presented in the plan as the re-creation
of the resource. Since Terraform is not directly responsible for the changes
made by the provisioners, deleting a stateful provisioning resource has no
effect and so application cleanup must be handled by some other process,
outside of Terraform.

## Example Usage

```
variable "revision_to_deploy" {
}

resource "stateful_provisioning" "app" {
    # Run deployment steps whenever the revision_to_deploy changes.
    state_key = "${var.revision_to_deploy}"

    provisioner "remote-exec" {
        inline = [
            "my-favorite-deployment-tool --revision=${var.revision_to_deploy}"
        ]

        connection {
            user = "deployment-user"
            host = "deployment-admin-host.example.com"
            key_file = "deployment_private_key"
        }
    }
}
```

Any provisioner available to Terraform may be used to specify actions to take
when the state is changed.

In many cases the stateful provisioning resource will depend on other
resources, such as the hosts on which the provisioning steps will be run.
As usual, these dependencies can either be implicitly defined by interpolating
values into the state key or provisioner configurations, or explicitly
defined using the ``depends_on`` configuration key.

## Argument Reference

The following argument is supported:

* `state_key` - (Required) Arbitrary string that uniquely identifies a
  particular application state. Interpolate into this string any data which
  can affect the outcome of the included provisioners.
