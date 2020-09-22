variable "middle_in" {
    type = "map"
    default = {
        eu-west-1 = "ami-12345"
        eu-west-2 = "ami-67890"
    }
}

module "inner" {
    source = "../inner"

    inner_in = "hello"
}

resource "null_resource" "middle_noop" {}

output "middle_out" {
    value = "${lookup(var.middle_in, "us-west-1")}"
}
