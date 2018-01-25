---
layout: "backend-types"
page_title: "Backend Type: local"
sidebar_current: "docs-backends-types-enhanced-local"
description: |-
  Terraform can store the state remotely, making it easier to version and work with in a team.
---

# local

**Kind: Enhanced**

The local backend stores state on the local filesystem, locks that
state using system APIs, and performs operations locally.

The local backend supports the concept of _sealing_ or writing state files (and backups) in an encrypted form using password-based encryption. Both `password_file_path` and `seal` must be set properly for encryption to occur, but can be altered individually to transition to / from an encrypted state.

Sealing local state is useful for scenarios where its appropriate to share state through version control (e.g., committed to a git repository). While sharing state through version control does not offer any locking protections against conccurrent execution, it does provide a basic security mechanism for small teams to share essential state. Every team should evaluate for itself whether doing so meets the team's security requirements. The implementation of this feature is analagous to and inspired by _vaults_ in [Ansible](http://docs.ansible.com/ansible/latest/vault.html). For many scenarios, if the use of Ansible vaults are acceptable, then using Terraform with sealed state should also be acceptable.

Transitioning to encrypted state can proceed with these steps:
* Set `password_file_path` in configuration
* Set `seal = true` in configuration
* Run `terraform init` to reinitialize the backend (any other command will prompt to do so anyway)
* Run `terraform apply` to apply any changes and write an encrypted state and an encrypted backup

Once state has been encrypted, it can be decrypted with a similar series of steps:
* Leave `password_file_path` set in configuration
* Set `seal = false` in configuration
* Run `terraform init` to reinitialize the backend (any other command will prompt to do so anyway)
* Run `terraform apply` to apply any changes and write a decrypted state and a decrypted backup

Note that because `password_file_path` supports referencing the home directory via `~`, then it's possible for keys to reside in a location separate from where actual Terraform configurations reside. Teams may find it useful to distribute keys separately through external, secure means, but require all team members to store such keys in identical locations so that backend configuration can reference the same path for all team members.

## Example Configuration

```hcl
terraform {
  backend "local" {
    path = "relative/path/to/terraform.tfstate"
    password_file_path = "~/.terraform/my_password.txt"
    seal = true
  }
}
```

## Example Reference

```hcl
data "terraform_remote_state" "foo" {
  backend = "local"

  config {
    path = "${path.module}/../../terraform.tfstate"
  }
}
```

## Configuration variables

The following configuration options are supported:

 * `path` - (Optional) The path to the `tfstate` file. This defaults to
   "terraform.tfstate" relative to the root module by default.
* `password_file_path` - (Optional) The path to a file containing the password to use if state should be written encrypted. The file is read as bytes, and thus can be text or other data suitable for use as a password. If not present, then state will not be encrypted when written. The use of `~` to reference the home directory is supported.
* `seal` - (Optional) A boolean indicating whether to write state in encrypted form or not. A value of `true` will write state in encrypted form, a value of `false` will write state in cleartext or decrypted form. The default is `false`. If present but no `password_file_path` is present, then has no effect.