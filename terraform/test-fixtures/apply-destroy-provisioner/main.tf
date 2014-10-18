resource "aws_instance" "foo" {
    id = "foo"

    provisioner "shell" {}
}
