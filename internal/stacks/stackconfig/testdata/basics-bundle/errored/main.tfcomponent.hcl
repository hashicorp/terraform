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


removed {
  // This is invalid, you must reference the for_each somewhere in the
  // from attribute if both are present.
  from = component.b["something"]

  for_each = ["a", "b"]

  source = "./"

  providers = {
    null = provider.null.a
  }
}

removed {
  // This is invalid, you must reference the for_each somewhere in the
  // from attribute if both are present.
  from = component.c

  for_each = ["a", "b"]

  source = "./"

  providers = {
    null = provider.null.a
  }
}