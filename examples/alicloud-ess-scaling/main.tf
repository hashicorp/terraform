data "alicloud_images" "ecs_image" {
  most_recent = true
  name_regex =  "^centos_6\\w{1,5}[64].*"
}

resource "alicloud_security_group" "sg" {
  name = "${var.security_group_name}"
  description = "tf-sg"
}

resource "alicloud_security_group_rule" "ssh-in" {
  type = "ingress"
  ip_protocol = "tcp"
  nic_type = "internet"
  policy = "accept"
  port_range = "22/22"
  priority = 1
  security_group_id = "${alicloud_security_group.sg.id}"
  cidr_ip = "0.0.0.0/0"
}

resource "alicloud_ess_scaling_group" "scaling" {
  min_size = "${var.scaling_min_size}"
  max_size = "${var.scaling_max_size}"
  scaling_group_name = "tf-scaling"
  removal_policies = "${var.removal_policies}"

}

resource "alicloud_ess_scaling_configuration" "config" {
  scaling_group_id = "${alicloud_ess_scaling_group.scaling.id}"
  enable = "${var.enable}"

  image_id = "${data.alicloud_images.ecs_image.images.0.id}"
  instance_type = "${var.ecs_instance_type}"
  io_optimized = "optimized"
  security_group_id = "${alicloud_security_group.sg.id}"
}