required_providers {
  null = {
    source  = "hashicorp/null"
    version = "3.2.1"
  }
}

variable "name" {
  type = string
}

variable "provider" {
  type = providerconfig(null)
}

component "a" {
  source = "../"

  inputs = {
    name = var.name
  }
  providers = {
    null = var.provider
  }
}

output "greeting" {
  type  = string
  value = component.a.greeting
}
