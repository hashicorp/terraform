variable "foo" {}

resource aws_instance "web" {
  lifecycle {
    ignore_changes = ["${var.foo}"]
  }
}
