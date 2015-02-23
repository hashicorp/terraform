resource "aws_instance" "foo" {}

resource "aws_instance" "bar" {
    provisioner "local-exec" {
        command = "${element(aws_instance.bar.*.private_ip, 0)}"
    }
}
