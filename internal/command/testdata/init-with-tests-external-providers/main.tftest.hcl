
// configure is not a "hashicorp" provider, so it won't be able to load
// this using the default behaviour. Terraform will need to look into the setup
// module to find the provider configuration.
provider "configure" {}

// testing is a "hashicorp" provider, so it can load this using the defaults
// even though not required provider block providers a definition for it.
provider "testing" {}

run "setup" {
  module {
    source = "./setup"
  }

  providers = {
    configure = configure
  }
}

run "test" {
  providers = {
    testing = testing
  }
}
