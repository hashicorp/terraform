variable "test_var" {
  default = "boop"
  sensitive = true
}

resource "test_instance" "test" {
  ami = var.test_var
}

output "test" {
  value = test_instance.test.ami
  sensitive = true
}
