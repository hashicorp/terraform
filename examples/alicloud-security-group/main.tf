resource "alicloud_security_group" "group" {
  name = "${var.short_name}"
  description = "New security group"
  vpc_id = "${var.vpc_id}"
}
