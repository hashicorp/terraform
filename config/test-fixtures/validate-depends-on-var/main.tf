variable "foo" {
    description = "bar"
}

resource aws_instance "web" {
    depends_on = ["${var.foo}"]
}
