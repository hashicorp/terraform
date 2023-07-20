# There is no provider in required_providers called "configured", so the version
# constraint should come from this configuration block.
provider "configured" {
  version = "~> 1.4"
}

run "setup" {
  module {
    source = "./setup"
  }
}
