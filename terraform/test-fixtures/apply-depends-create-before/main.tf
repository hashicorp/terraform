
resource "aws_instance" "web" {
    require_new = "ami-new"
    lifecycle {
        create_before_destroy = true
    }
}

resource "aws_instance" "lb" {
    instance = "${aws_instance.web.id}"
}
