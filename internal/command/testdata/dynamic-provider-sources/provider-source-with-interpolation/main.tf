variable "provider_ns" {
  type  = string
  const = true
}

terraform {
  required_providers {
    test = {
      source = "${var.provider_ns}/test"
    }
  }
}
