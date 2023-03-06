// This should result in installing hashicorp/foo
provider null {}

// This will try to install hashicorp/http, fail, and then suggest
// terraform-providers/http
provider http {}

// This will try to install hashicrop/frob, fail, find no suggestions, and
// result in an error
provider tls {}

module "some-baz-stuff" {
  source = "./child"
}

module "dicerolls" {
  source = "acme/bar/random"
}
