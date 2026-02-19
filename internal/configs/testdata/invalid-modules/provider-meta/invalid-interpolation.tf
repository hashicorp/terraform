terraform {
  provider_meta "my-provider" {
    hello = var.name
  }
}

variable "name" {
  type = string
}

