variable "count" {
    default = 2
}

module "foo" {
    count = "${var.count}"
    source = "./foo"
}

resource "aws_volume_attachment" "foo" {
    count = "${var.count}"
    instance_id = "${module.foo.instance_ids[count.index]}-baz"
    volume_id = "${module.foo.volume_ids[count.index]}-baz"
}
