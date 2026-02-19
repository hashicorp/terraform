variable "foo" {
  sensitive = true
}

resource "aws_instance" "foo" {
  ami = var.foo

  lifecycle {
    ignore_changes = [ami]
  }
}
