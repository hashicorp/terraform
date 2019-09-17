resource "aws_instance" "bar" {
    provisioner "shell" {
        foo = "bar"
    }
}
