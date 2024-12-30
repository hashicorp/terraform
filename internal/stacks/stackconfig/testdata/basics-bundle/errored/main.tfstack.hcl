required_providers {
  null = {
    source  = "hashicorp/null"
    version = "3.2.1"
  }
}

provider "null" "a" {}

component "a" {
  source = "./"

  inputs = {
      name = var.name
  }

  providers = {
      null = provider.null.a
  }
}

removed {
  // This is invalid, you can't reference the whole component like this if
  // the target component is still in the config.
  from = component.a

  source = "./"

  providers = {
    null = provider.null.a
  }
}
