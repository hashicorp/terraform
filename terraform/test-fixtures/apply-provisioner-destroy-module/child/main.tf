variable "key" {}

resource "aws_instance" "foo" {
    foo = "bar"

    provisioner "shell" {
        foo  = "${var.key}"
        when = "destroy"
    }
}
