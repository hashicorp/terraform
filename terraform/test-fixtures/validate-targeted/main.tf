resource "aws_instance" "foo" {
    num = "2"
    provisioner "shell" {
       command = "echo hi"
    }
}

resource "aws_instance" "bar" {
    foo = "bar"
    provisioner "shell" {
       command = "echo hi"
    }
}
