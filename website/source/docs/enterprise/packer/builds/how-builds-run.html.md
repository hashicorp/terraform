---
title: "How Packer Builds Run in Atlas"
---

# How Packer Builds Run in Atlas

This briefly covers the internal process of running builds in Atlas. It's
not necessary to know this information, but may be valuable to
help understand implications of running in Atlas or debug failing
builds.

### Steps of Execution

1. A Packer template and directory of files is uploaded via Packer Push or GitHub
1. Atlas creates a version of the build configuration and waits for the upload
to complete. At this point, the version will be visible in the UI even if the upload has
not completed
1. Once the upload finishes, Atlas queues the build. This is potentially
split across multiple machines for faster processing
1. In the build environment, the package including the files and Packer template
are downloaded
1. `packer build` is run against the template in the build environment
1. Logs are streamed into the UI and stored
1. Any artifacts as part of the build are then uploaded via the public
Atlas artifact API, as they would be if Packer was executed locally
1. The build completes, the environment is teared down and status
updated within Atlas

