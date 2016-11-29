variable "param" {
  type = "list"
}

resource "aws_instance" "test" {
  count    = "2"
  thing = "doesnt"
}

output "out_from_splat" {
  value = ["${aws_instance.test.*.thing}"]
}
