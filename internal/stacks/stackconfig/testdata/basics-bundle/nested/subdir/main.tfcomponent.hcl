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

removed {
  from = component.b

  source = "../"
  providers = {
    null = var.provider
  }

  lifecycle {
    destroy = true
  }
}

output "greeting" {
  type  = string
  value = component.a.greeting
}
