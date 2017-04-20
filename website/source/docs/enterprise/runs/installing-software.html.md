---
layout: "enterprise"
page_title: "Installing Software - Runs - Terraform Enterprise"
sidebar_current: "docs-enterprise-runs-installing"
description: |-
  Installing custom software on the Terraform Runners.
---

# Installing Custom Software

The machines that run Terraform exist in an isolated environment and are
destroyed on each use. In some cases, it may be necessary to install certain
software on the Terraform runner, such as a configuration management tool like
Chef, Puppet, Ansible, or Salt.

The easiest way to install software on the Packer builder is via the
`local-exec` provisioner. This will execute commands on the host machine running
Terraform.

```hcl
resource "null_resource" "local-software" {
  provisioner "local-exec" {
    command = <<EOH
sudo apt-get update
sudo apt-get install -y ansible
EOH
  }
}
```

Please note that nothing is persisted between Terraform runs, so you will need
to install custom software on each run.

The Packer builders run the latest version of Ubuntu LTS.
