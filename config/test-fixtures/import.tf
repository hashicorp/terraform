import "import/one.tf";

variable "foo" {
    default = "bar";
    description = "bar";
}

resource "aws_security_group" "web" {}
