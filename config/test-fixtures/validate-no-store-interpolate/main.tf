variable "foo" {}

resource aws_instance "web" {
  lifecycle {
    no_store = ["${var.foo}"]
  }
}
