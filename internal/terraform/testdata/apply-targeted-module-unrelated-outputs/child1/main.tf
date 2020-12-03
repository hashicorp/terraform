variable "instance_id" {
}

output "instance_id" {
  # The instance here isn't targeted, so this output shouldn't get updated.
  # But it already has an existing value in state (specified within the
  # test code) so we expect this to remain unchanged afterwards.
  value = "${aws_instance.foo.id}"
}

output "given_instance_id" {
  value = "${var.instance_id}"
}

resource "aws_instance" "foo" {
  foo = "${var.instance_id}"
}
