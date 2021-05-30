resource "aws_instance" "foo" {
    count = 3

    provisioner "local-exec" {
        command = "echo ${aws_instance.foo.0.id} ${aws_instance.foo.1.id} ${aws_instance.foo.2.id}"
    }
}
