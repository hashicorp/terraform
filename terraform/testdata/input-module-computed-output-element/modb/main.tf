resource "aws_instance" "test" {
  count = 3
}

output "computed_list" {
  value = ["${aws_instance.test.*.id}"]
}
