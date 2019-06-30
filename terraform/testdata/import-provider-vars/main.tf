variable "foo" {}

provider "aws" {
  foo = "${var.foo}"
}
