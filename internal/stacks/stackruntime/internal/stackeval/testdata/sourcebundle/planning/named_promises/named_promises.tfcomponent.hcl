
required_providers {
  happycloud = {
    source  = "example.com/test/happycloud"
    version = "1.0.0"
  }
}

variable "in" {
  type = string
}

provider "happycloud" "main" {
}

stack "child" {
  source = "./child"

  inputs = {
    in = var.in
  }
}

component "foo" {
  source = "./"

  providers = {
    happycloud = provider.happycloud.main
  }
}

output "out" {
  type  = string
  value = var.in
}
