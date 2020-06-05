variable "test_var" {
  default = "bar"
}
resource "test_instance" "test" {
  ami   = var.test_var
  count = 3
}

output "test" {
  value = var.test_var
}
