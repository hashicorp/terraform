resource "aws_instance" "foo" {
    foo = "bar"

    connection {
        host = "${self.foo}"
    }

    provisioner "shell" {
        value = "${self.foo}"
    }
}
