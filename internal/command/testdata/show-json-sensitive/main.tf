provider "test" {
  region = "somewhere"
}

variable "test_var" {
  default   = "bar"
  sensitive = true
}

resource "test_instance" "test" {
  // this variable is sensitive
  ami = var.test_var
  // the password attribute is sensitive in the showFixtureSensitiveProvider schema.
  password = "secret"
  count    = 3
}

output "test" {
  value     = var.test_var
  sensitive = true
}
