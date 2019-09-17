provider "aws" {
  alias = "foo"
}

// removed module configuration referencing aws.foo, which was passed in by the
// root module
