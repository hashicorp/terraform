---
title: "Installing Software"
---

# Installing Software

Please review the [Packer Build Environment](/help/packer/builds/build-environment)
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
