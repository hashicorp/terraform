resource "aws_instance" "foo" {
    cwd = "${path.cwd}/barpath"
    module = "${path.module}/foopath"
    root = "${path.root}/barpath"
}
