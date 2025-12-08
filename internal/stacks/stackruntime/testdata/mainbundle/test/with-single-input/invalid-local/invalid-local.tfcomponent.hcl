required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "default" {}

variable "auto_branch_creation_config" {
  type = object({
    basic_auth_credentials        = optional(string)
    build_spec                    = optional(string)
    enable_auto_build             = optional(bool)
    enable_basic_auth             = optional(bool)
    enable_performance_mode       = optional(bool)
    enable_pull_request_preview   = optional(bool)
    environment_variables         = optional(map(string))
    secrets                       = optional(map(string))
    framework                     = optional(string)
    pull_request_environment_name = optional(string)
    stage                         = optional(string)
  })
  description = "The automated branch creation configuration for the Amplify app"
  default     = null
}

locals {
  # BROKEN VERSION: Treats the config as if it were a map
  # This triggers the "unknown value" error in Terraform
  auto_branch_creation_config = var.auto_branch_creation_config != null ? {
    for k, v in var.auto_branch_creation_config : k => merge(
      { for attr, val in v : attr => val if attr != "secrets" },
      v.environment_variables != null || v.secrets != null ? {
        environment_variables = merge(
          coalesce(v.environment_variables, {}),
          { for sk, sv in coalesce(v.secrets, {}) : sk => sv }
        )
      } : {}
    )
  } : null
}

component "self" {
  source = "../"

  providers = {
    testing = provider.testing.default
  }

  inputs = {
    id   = "static-id"
    input = jsonencode(local.auto_branch_creation_config)
  }
}
