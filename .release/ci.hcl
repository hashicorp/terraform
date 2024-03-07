# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: BUSL-1.1

schema = "2"

project "terraform" {
  // the team key is not used by CRT currently
  team = "terraform"
  slack {
    notification_channel = "C011WJ112MD"
  }
  github {
    organization = "hashicorp"
    repository = "terraform"

    release_branches = [
      "main",
      "release/**",
      "v**.**",
    ]
  }
}

event "build" {
  depends = ["merge"]
  action "build" {
    organization = "hashicorp"
    repository = "terraform"
    workflow = "build"
  }
}

// Read more about what the `prepare` workflow does here:
// https://hashicorp.atlassian.net/wiki/spaces/RELENG/pages/2489712686/Dec+7th+2022+-+Introducing+the+new+Prepare+workflow
event "prepare" {
  depends = ["build"]

  action "prepare" {
    organization = "hashicorp"
    repository   = "crt-workflows-common"
    workflow     = "prepare"
    depends      = ["build"]
  }

  notification {
    on = "fail"
  }
}

## These are promotion and post-publish events
## they should be added to the end of the file after the verify event stanza.

event "trigger-staging" {
// This event is dispatched by the bob trigger-promotion command
// and is required - do not delete.
}

event "promote-staging" {
  depends = ["trigger-staging"]
  action "promote-staging" {
    organization = "hashicorp"
    repository = "crt-workflows-common"
    workflow = "promote-staging"
    config = "release-metadata.hcl"
  }

  notification {
    on = "always"
  }
}

event "trigger-production" {
// This event is dispatched by the bob trigger-promotion command
// and is required - do not delete.
}

event "promote-production" {
  depends = ["trigger-production"]
  action "promote-production" {
    organization = "hashicorp"
    repository = "crt-workflows-common"
    workflow = "promote-production"
  }

  promotion-events  {
    update-ironbank = true
    post-promotion {
      organization = "hashicorp"
      repository = "terraform-releases"
      workflow = "crt-hook-tfc-upload"
    }
  }

  notification {
    on = "always"
  }
}
