variable "key" {}

resource "aws_instance" "foo" {
    foo = "bar"

    provisioner "shell" {
        command = "${var.key}"
        when = "destroy"
    }
}
