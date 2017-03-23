---
layout: "packer"
page_title: "Installing Software with Packer"
sidebar_current: "docs-enterprise-packer-builds-installing"
description: |-
  Installing software with Packer.
---

# Installing Software

Please review the [Packer Build Environment](/docs/enterprise/packer/builds/build-environment.html)
specification for important information on isolation, security, and hardware
limitations before continuing.

In some cases, it may be necessary to install custom software to build your
artifact using Packer. The easiest way to install software on the Packer builder
is via the `shell-local` provisioner. This will execute commands on the host
machine running Packer.

    {
      "provisioners": [
        {
          "type": "shell-local",
          "command": "sudo apt-get install -y customsoftware"
        }
      ]
    }

Please note that nothing is persisted between Packer builds, so you will need
to install custom software on each run.

The Packer builders run the latest version of Ubuntu LTS.
