required_providers {
  terraform = {
    source = "terraform.io/builtin/terraform"
  }
}

provider "terraform" "default" {
}

component "self" {
  source = "./"

  providers = {
    terraform = provider.terraform.default
  }
}

output "obj" {
  type = object({
    input  = string
    output = string
  })
  value = component.self
}
