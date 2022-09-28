variable "test_var" {
  default = "bar"
}

// There is a single instance in state. The plan will add a resource.
resource "test_instance" "test" {
  ami   = var.test_var
  count = 2
}

output "test" {
  value = var.test_var
}
