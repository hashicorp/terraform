resource "aws_instance" "foo" {}

resource "aws_instance" "bar" {
    connection {
        host = "${element(aws_instance.foo.*.private_ip, 0)}"
    }

    provisioner "local-exec" {}
}
