provider "alicloud" {
  region = "${var.region}"
}

module "vpc" {
  availability_zones = "${var.availability_zones}"
  source = "../alicloud-vpc"
  short_name = "${var.short_name}"
  region = "${var.region}"
}

module "security-groups" {
  source = "../alicloud-vpc-cluster-sg"
  short_name = "${var.short_name}"
  vpc_id = "${module.vpc.vpc_id}"
}

module "control-nodes" {
  source = "../alicloud-ecs-vpc"
  count = "${var.control_count}"
  role = "control"
  datacenter = "${var.datacenter}"
  ecs_type = "${var.control_ecs_type}"
  ecs_password = "${var.ecs_password}"
  disk_size = "${var.control_disk_size}"
  ssh_username = "${var.ssh_username}"
  short_name = "${var.short_name}"
  availability_zones = "${module.vpc.availability_zones}"
  security_groups = ["${module.security-groups.control_security_group}"]
  vswitch_id = "${module.vpc.vswitch_ids}"
  internet_charge_type = "${var.internet_charge_type}"
}

module "edge-nodes" {
  source = "../alicloud-ecs-vpc"
  count = "${var.edge_count}"
  role = "edge"
  datacenter = "${var.datacenter}"
  ecs_type = "${var.edge_ecs_type}"
  ecs_password = "${var.ecs_password}"
  ssh_username = "${var.ssh_username}"
  short_name = "${var.short_name}"
  availability_zones = "${module.vpc.availability_zones}"
  security_groups = ["${module.security-groups.worker_security_group}"]
  vswitch_id = "${module.vpc.vswitch_ids}"
  internet_charge_type = "${var.internet_charge_type}"
}

module "worker-nodes" {
  source = "../alicloud-ecs-vpc"
  count = "${var.worker_count}"
  role = "worker"
  datacenter = "${var.datacenter}"
  ecs_type = "${var.worker_ecs_type}"
  ecs_password = "${var.ecs_password}"
  ssh_username = "${var.ssh_username}"
  short_name = "${var.short_name}"
  availability_zones = "${module.vpc.availability_zones}"
  security_groups = ["${module.security-groups.worker_security_group}"]
  vswitch_id = "${module.vpc.vswitch_ids}"
  internet_charge_type = "${var.internet_charge_type}"
}