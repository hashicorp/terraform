component "a" {
  source = "./component"

  inputs = {
    name = var.name
  }
  providers = {
    null = var.provider
  }
}

provider "null" "a" {
}

locals {
  sound = "bleep bloop"
}
