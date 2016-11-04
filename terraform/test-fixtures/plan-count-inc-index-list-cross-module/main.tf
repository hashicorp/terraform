variable "count" {
    default = 2
}

module "foo" {
    count = "${var.count}"
    source = "./foo"
}

resource "aws_ebs_volume" "foo" {
    count = "${var.count}"
    foo = "foo"
}

resource "aws_volume_attachment" "foo" {
    count = "${var.count}"
    instance_id = "${module.foo.instance_ids[count.index]}-baz"
    volume_id = "${aws_ebs_volume.foo.*.id[count.index]}-baz"
}
