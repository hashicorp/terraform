---
title: "Collaborating on Terraform Remote State in Atlas"
---

# Collaborating on Terraform Remote State in Atlas

Atlas is one of a few options to store [remote state](/help/terraform/state).

Remote state gives you the ability to version and collaborate on Terraform changes. It
stores information about the changes Terraform makes based on configuration.

In order to collaborate safely on remote state, we recommend
[creating an organization](/help/organizations/create) to manage teams of users.

Then, following a [remote state push](/help/terraform/state) you can view state versions
in the changes tab of the [environment](/help/glossary#environment) created under the same name
as the remote state.
