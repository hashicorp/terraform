---
layout: "pass"
page_title: "Provider: Pass"
sidebar_current: "docs-pass-index"
description: |-
  The Pass provider allows Terraform to read from, write to, and configure Pass Password-store
---

# Pass Provider

The Pass provider allows Terraform to read from, write to, and configure
[Pass](https://www.passwordstore.org/).

~> **Important** Interacting with Pass from Terraform causes any secrets
that you read and write to be persisted in both Terraform's state file
*and* in any generated plan files. For any Terraform module that reads or
writes Pass secrets, these files should be treated as sensitive and
protected accordingly.

## Using Pass credentials in Terraform configuration

Most Terraform providers require credentials to interact with a third-party
service that they wrap. This provider allows such credentials to be obtained
from Pass, which means that operators or systems running Terraform need
only access to a suitably-privileged Pass password-store in order to
temporarily lease the credentials for other providers.

Currently Terraform has no mechanism to redact or protect secrets that
are returned via data sources, so secrets read via this provider will be
persisted into the Terraform state, into any plan files, and in some cases
in the console output produced while planning and applying. These artifacts
must therefore all be protected accordingly.

## Provider Arguments

The provider configuration block accepts the following arguments.
In most cases it is recommended to set them via the indicated environment
variables in order to keep credential information out of the configuration.

* `store_dir` - (Optional) Overrides the default password storage directory.
  May be set via the `PASSWORD_STORE_DIR` environment variable.

## Example Usage

```
provider "pass" {
}

data "pass_password" "example" {
  path = "secret/foo"
}
```


