resource "aws_instance" "no_count" {
}

resource "aws_instance" "count" {
  count = 1
}
