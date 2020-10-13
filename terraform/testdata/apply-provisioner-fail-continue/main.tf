resource "aws_instance" "foo" {
    foo = "bar"

    provisioner "shell" {
        on_failure = "continue"
    }
}
