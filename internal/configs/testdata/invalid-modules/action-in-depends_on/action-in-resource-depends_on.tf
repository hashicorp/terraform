action "aws_action" "example" {
}

resource "aws_instance" "web" {
  depends_on = [action.aws_action.example]
  ami        = "ami-1234"
  security_groups = [
    "foo",
    "bar",
  ]
}
