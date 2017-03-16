---
title: "Pushing Terraform Remote State to Atlas"
---

# Pushing Terraform Remote State to Atlas

Atlas is one of a few options to store [remote state](/help/terraform/state).

Remote state gives you the ability to version and collaborate on Terraform changes. It
stores information about the changes Terraform makes based on configuration.

To use Atlas to store remote state, you'll first need to have the
`ATLAS_TOKEN` environment variable set and run the following command.

    $ terraform remote config -backend-config="name=%{DEFAULT_USERNAME}/product"
