# This is a configuration with HCP Terraform mode activated but without
# a workspaces block, which should trigger an "Invalid workspaces configuration"
# error during initialization. This is used to test that the diagnostic
# formatting correctly processes color tokens in the error detail message.

terraform {
  cloud {
    organization = "PLACEHOLDER"
  }
}
