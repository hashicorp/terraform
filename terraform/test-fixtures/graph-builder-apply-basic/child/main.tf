resource "aws_instance" "create" {
    provisioner "exec" {}
}

resource "aws_instance" "other" {
    value = "${aws_instance.create.id}"
}
