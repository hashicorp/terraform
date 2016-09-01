resource "aws_instance" "foo" {
    count = 3

    provisioner "shell" {
        command = "${self.count}"
    }
}
