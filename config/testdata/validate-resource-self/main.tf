resource "aws_instance" "foo" {
    foo = "${self.bar}"
}
