---
title: "Schedule Periodic Builds in Atlas"
---

# Schedule Periodic Builds in Atlas

Atlas can automatically run a Packer build and
create artifacts on a specified schedule. This option is disabled by default and can be enabled by an
organization owner on a per-[environment](/help/glossary#environment) basis.

On the specified interval, Atlas will automatically queue a build that
runs Packer for you, creating any artifacts and sending the appropriate
notifications.

If your artifacts are used in any other environments and you have activated
the plan on aritfact upload feature, this may also queue Terraform
plans.

This feature is useful for maintenance of images and automatic updates,
or to build nightly style images for staging or development environments.

## Enabling Periodic Builds

To enable periodic builds for a build, visit the build settings page in
Atlas and select the desired interval and click the save button to
persist the changes. An initial build may immediately run, depending
on the history, and then will automatically build at the specified interval.

If you have run a build separately, either manually or triggered from GitHub
or Packer configuration version uploads, Atlas will not queue a new
build until the alloted time after the manual build ran. This means that
Atlas simply ensures that a build has been executed at the specified schedule.
