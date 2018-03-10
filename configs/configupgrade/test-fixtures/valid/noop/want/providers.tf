
terraform {
  required_version = ">= 0.7.0, <0.13.0"

  backend "local" {
    path = "foo.tfstate"
  }
}

provider "test" {
}
