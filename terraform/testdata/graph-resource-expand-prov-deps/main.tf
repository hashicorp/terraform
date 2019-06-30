resource "aws_instance" "web" {
    count = 3

    provisioner "remote-exec" {
        inline = ["echo ${aws_instance.web.0.foo}"]
    }
}
