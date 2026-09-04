# There is no provider in required_providers called "configured", so we won't
# have a version constraint for it.
provider "configured" {}

run "setup" {
  module {
    source = "./setup"
  }
}
