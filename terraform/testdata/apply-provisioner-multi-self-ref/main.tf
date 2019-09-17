resource "aws_instance" "foo" {
    count = 3
    foo = "number ${count.index}"

    provisioner "shell" {
        command = "${self.foo}"
    }
}
