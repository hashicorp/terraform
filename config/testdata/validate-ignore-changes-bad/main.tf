variable "foo" {
  default = "ami-abcd1234"
}

variable "bar" {
  default = "t2.micro"
}

provider "aws" {
  access_key = "foo"
  secret_key = "bar"
}

resource aws_instance "web" {
  ami           = "${var.foo}"
  instance_type = "${var.bar}"

  lifecycle {
    ignore_changes = ["ami", "instance*"]
  }
}
