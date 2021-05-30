variable "test_var" {
  default = "bar-var"
}

output "test" {
  value = var.test_var
}

resource "test_instance" "test" {
  ami = var.test_var
}
