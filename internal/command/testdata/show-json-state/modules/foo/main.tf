variable "test_var" {
  default = "foo-var"
}

resource "test_instance" "test" {
  ami   = var.test_var
  count = 1
}

output "test" {
  value = var.test_var
}

provider "test" {}
