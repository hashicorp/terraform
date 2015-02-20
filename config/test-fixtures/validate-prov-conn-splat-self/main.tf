resource "aws_instance" "bar" {
    connection {
        host = "${element(aws_instance.bar.*.private_ip, 0)}"
    }

    provisioner "local-exec" {}
}
