resource "aws_instance" "bar" {
    num = "2"

    provisioner "shell" {
        foo = "${aws_instance.bar.num}"
    }
}
