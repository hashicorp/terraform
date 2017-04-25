---
layout: "enterprise"
page_title: "Running - Packer Builds - Terraform Enterprise"
sidebar_current: "docs-enterprise-packerbuilds-runbuilds"
description: |-
  This briefly covers the internal process of running builds in Terraform Enterprise.
---

# How Packer Builds Run in Terraform Enterprise

This briefly covers the internal process of running builds in Terraform
Enterprise. It's not necessary to know this information, but may be valuable to
help understand implications of running or debugging failing builds.

### Steps of Execution

1. A Packer template and directory of files is uploaded via Packer Push or
GitHub

2. Terraform Enterprise creates a version of the build configuration and waits
for the upload to complete. At this point, the version will be visible in the UI
even if the upload has not completed

3. Once the upload finishes, the build is queued. This is potentially split
across multiple machines for faster processing

4. In the build environment, the package including the files and Packer template
are downloaded

5. `packer build` is run against the template in the build environment

6. Logs are streamed into the UI and stored

7. Any artifacts as part of the build are then uploaded via the public artifact
API, as they would be if Packer was executed locally

8. The build completes, the environment is teared down and status updated
