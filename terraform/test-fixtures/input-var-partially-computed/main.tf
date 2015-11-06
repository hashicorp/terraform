resource "aws_instance" "foo" { }
resource "aws_instance" "bar" { }

module "child" {
  source = "./child"
  in = "one,${aws_instance.foo.id},${aws_instance.bar.id}"
}
