resource "aws_instance" "foo" {
  num         = "2"
  provisioner "shell"     {}
}

resource "aws_instance" "bar" {
  foo         = "bar"
  provisioner "shell"     {}
}
