variable "instance_id" {}

output "instance_id" {
  value = "${var.instance_id}"
}

resource "test_object" "foo" {
  triggers = {
    instance_id = "${var.instance_id}"
  }
}
