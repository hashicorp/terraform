---
layout: "docs"
page_title: "Provisioner: ansible-local"
sidebar_current: "docs-provisioners-ansible-local"
description: |-
  The `ansible-local` provisioner runs Ansible on a resource.
---


# Ansible-local Provisioner

The `ansible-local` provisioner runs Ansible locally to a remote
resource.

## Requirements

The `ansible-local` provisioner requires that Ansible is already installed on the remote machine. It is common practise
to used the [remote-exec provisioner](/docs/provisioners/remote-exec.html) before the Ansible provisioner to do this.

## Example usage

```hcl
resource "aws_instance" "web" {
  # ...

  provisioner "ansible-local" {
    playbook_file = "${path.module}/ansible/playbook.yml"
    role_paths = ["${path.module}/ansible/roles/webserver"]
    extra_arguments = ["--verbose"]
  }
}
```

## Argument Reference

The following arguments are supported:

* `command (string)` - (Optional) The command to invoke Ansible (default is
`ANSIBLE_FORCE_COLOR=1 PYTHONUNBUFFERED=1 ansible-playbook`).

* `extra_arguments (array)` - (Optional) An array of extra arguments to pass to the Ansible command. These arguments
will be passed through a shell and arguments should be quoted accordingly.

* `group_vars (string)` - (Optional) A path to the directory containing Ansible group variables on your local system to
be copied to the remote machine.

* `host_vars (string)` - (Optional) A path to the directory containing Ansible host variables on your local system to
be copied to the remote machine.

* `galaxy_command (string)` - (Optional) The command to invoke the
[ansible-galaxy cli](http://docs.ansible.com/ansible/galaxy.html#the-ansible-galaxy-command-line-tool) (defaults to
`ansible-galaxy`). [ansible-galaxy cli](http://docs.ansible.com/ansible/galaxy.html#the-ansible-galaxy-command-line-tool)
is only required if `galaxy_file` is defined.

* `galaxy_file (string)` - (Optional) A requirements file which provides a way to install roles with the
[ansible-galaxy cli](http://docs.ansible.com/ansible/galaxy.html#the-ansible-galaxy-command-line-tool) on the remote machine.

* `playbook_directory (string)` - (Optional) A path to the complete Ansible directory structure on your local system to
be copied to the remote machine as the `staging_directory` before all other files and directories.

* `playbook_file (string)` - (Required) The playbook file to be executed by Ansible. This file will be uploaded to the
remote machine. 

* `playbook_paths (array)` - (Optional) An array of directories of playbook files on your local system. These will be
uploaded to the remote machine under `staging_directory/playbooks`.

* `inventory_file (string)` - (Optional) The inventory file to be used by Ansible. This file will be be uploaded to the
remote machine. Conflicts with `inventory_groups`.

* `inventory_groups (array)` - (Optional) A list of groups that the host `127.0.0.1` should be a member of in the 
generated Ansible inventory file. Conflicts with `inventory_file`.

* `role_paths (array)` - (Optional) An array of paths to role directories on your local system. These will be uploaded
to the remote machine under `staging_directory/roles`.

* `staging_directory (string)` - (Optional) The directory where all configuration of Ansible will be placed. By default
this is `/tmp/terraform-provisioner-ansible-local/<uuid>` where `<uuid>` is replace with a unique ID. This directory
doesn't need to exist but must have proper permissions so that files can be written to and directories created. 
