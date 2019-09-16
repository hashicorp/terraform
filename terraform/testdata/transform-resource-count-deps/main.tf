resource "aws_instance" "foo" {
    count = 2

    provisioner "local-exec" {
        command = "echo ${aws_instance.foo.0.id}"
        other = "echo ${aws_instance.foo.id}"
    }
}
