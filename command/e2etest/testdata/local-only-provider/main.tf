# The purpose of this test is to refer to a provider whose address contains
# a hostname that is only used for namespacing purposes and doesn't actually
# have a provider registry deployed at it.
#
# A user can install such a provider in one of the implied local filesystem
# directories and Terraform should accept that as the selection for that
# provider without producing any errors about the fact that example.com
# does not have a provider registry.
#
# For this test in particular we're using the "vendor" directory that is
# the documented way to include provider plugins directly inside a
# configuration uploaded to Terraform Cloud, but this functionality applies
# to all of the implicit local filesystem search directories.

terraform {
  required_providers {
    happycloud = {
      source = "example.com/awesomecorp/happycloud"
    }
  }
}
