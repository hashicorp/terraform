variable "instance_id" {
}

output "instance_id" {
  value = "${var.instance_id}"
}

resource "null_resource" "foo" {
  triggers = {
    instance_id = "${var.instance_id}"
  }
}
