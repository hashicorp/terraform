required_providers {
  # Built-in providers can be omitted.
}

provider "terraform" "x" {}

component "a" {
  source = "./component"

  inputs = {
    name = "test-name"
  }

  providers = {
    terraform = provider.terraform.x
  }
}

output "greeting" {
  type  = string
  value = component.a.greeting
}
