required_providers {
  null = {
    source  = "hashicorp/null"
    version = "3.2.1"
  }
}

variable "name" {
  type = string
}

provider "null" "a" {}

component "a" {
  source = "./component"

  inputs = {
    name = var.name
  }
  providers = {
    null = var.provider
  }
}

removed {
  from = component.b

  source = "./component"
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
