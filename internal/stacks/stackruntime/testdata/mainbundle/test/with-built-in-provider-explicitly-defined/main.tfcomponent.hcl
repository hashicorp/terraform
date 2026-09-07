required_providers {
  terraform = {
    source  = "terraform.io/builtin/terraform"
  }
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
