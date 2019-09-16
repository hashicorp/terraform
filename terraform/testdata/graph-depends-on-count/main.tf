resource "aws_instance" "web" {}

resource "aws_instance" "db" {
	depends_on = ["aws_instance.web"]
	count = 2
}
