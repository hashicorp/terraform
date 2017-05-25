resource "aws_instance" "create" {
	bar = "abc"
}

resource "aws_instance" "other" {
	var = "${aws_instance.create.id}"
    foo = "${aws_instance.create.bar}"
}
