resource "aws_instance" "bar" {
    foo = "bar"
    provisioner "shell" {}
}
