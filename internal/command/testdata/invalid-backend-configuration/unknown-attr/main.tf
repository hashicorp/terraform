terraform {
  backend "local" {
    path    = "foobar.tfstate"
    unknown = "this isn't in the local backend's schema" # Should trigger an error
  }
}
