resource "aws_instance" "abandon" {
  ami = "foo"
  lifecycle {
    abandon_on_destroy = true
  }
}

resource "aws_instance" "no_lifecycle" {
  ami = "foo"
}
