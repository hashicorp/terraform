required_providers {
  # Built-in providers can be omitted.
}

provider "terraform" "x" {}

component "a" {
  source = "./component"

  inputs = {
    name = var.name
  }

  providers = {
    x = var.provider
  }
}

output "greeting" {
  type  = string
  value = component.a.greeting
}
