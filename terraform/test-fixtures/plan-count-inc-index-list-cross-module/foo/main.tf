variable "count" {}

output "instance_ids" {
    value = ["aws_instance.foo.*.id"]
}

output "volume_ids" {
    value = ["aws_ebs_volume.foo.*.id"]
}

resource "aws_instance" "foo" {
    count = "${var.count}"
    foo = "foo"
}

resource "aws_ebs_volume" "foo" {
    count = "${var.count}"
    foo = "foo"
}
