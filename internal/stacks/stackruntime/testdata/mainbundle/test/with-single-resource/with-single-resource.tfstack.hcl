required_providers {
  terraform = {
    source = "terraform.io/builtin/terraform"
  }
}

provider "terraform" "default" {
}

component "self" {
  # FIXME: The source address parser we're using for components seems to be
  # requiring us to write "./." instead of "./" here, claiming the former
  # is the canonical form. A bug in go-slug's sourceaddrs package?
  source = "./."

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
