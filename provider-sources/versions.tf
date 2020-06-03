terraform {
  required_version = ">= 0.13.0"
  required_providers {
    aws = {
      # We don't strictly need to set "source" for this one because source
      # addresses are optional for providers under "hashicorp", but we
      # recommend setting them anyway in future to be explicit.
      source  = "hashicorp/aws"
      version = "~> 2.60.0"
    }

    testing = {
      # This is a third-party provider published experimentally as part of
      # the public registry beta, just to illustrate selecting third-party
      # providers. (Don't use this provider in production yet!)
      source  = "apparentlymart/testing"
      version = "0.0.1"
    }

    terraform = {
      # The terraform provider is built in to Terraform, so it has a different
      # source address. Terraform already uses this address by default for a
      # provider called "terraform", so specifying isn't necessary but we can
      # do it anyway for completeness.
      source = "terraform.io/builtin/terraform"
    }
  }
}

# In a normal configuration we'd then write provider, resource, and data blocks
# using the providers declared above. We don't have any here because this
# example is primarily to test with "terraform init", but please add other
# configuration here to try this out with throwaway infrastructure objects!
