terraform {
  backend "local" {
    path = "foobar.tfstate"
    path = "foobar2.tfstate" # Triggers a HCL-level error.
  }
}