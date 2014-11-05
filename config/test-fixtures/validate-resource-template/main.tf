resource_template "web" {
  count = 10
}

resource "aws_instance" "web" {
  resource_template = "web"
}