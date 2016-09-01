variable "count" {
    default = 2
}

resource "aws_instance" "foo" {
    count = "${var.count}"
    foo = "foo"
}

resource "aws_instance" "bar" {
    count = "${var.count}"
    foo = "${element(aws_instance.foo.*.foo,count.index)}-bar"
}

resource "aws_ebs_volume" "foo" {
    count = "${var.count}"
    foo = "foo"
}

resource "aws_volume_attachment" "foo" {
    count = "${var.count}"
    instance_id = "${element(aws_instance.foo.*.id,count.index)}"
    volume_id = "${element(aws_ebs_volume.foo.*.id,count.index)}"
}
