
required_providers {
  test = {
    source  = "example.com/test/test"
    version = "1.0.0"
  }
}

variable "name" {
  type = string
}

provider "test" "foo" {
  config {
    name = var.name
  }
}
