variable "list" {
  type = "list"
}

resource "aws_instance" "bar" {
	count = "${var.list[0]}"
}
