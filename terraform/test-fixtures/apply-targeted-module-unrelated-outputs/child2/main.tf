resource "aws_instance" "foo" {
}

output "instance_id" {
  # Even though we're targeting just the resource a bove, this should still
  # be populated because outputs are implicitly targeted when their
  # dependencies are
  value = "${aws_instance.foo.id}"
}
