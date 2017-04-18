variable "security_group_name" {
  default = "tf-sg"
}

variable "scaling_min_size" {
  default = 1
}

variable "scaling_max_size" {
  default = 1
}

variable "enable" {
  default = true
}

variable "removal_policies" {
  type    = "list"
  default = ["OldestInstance", "NewestInstance"]
}

variable "ecs_instance_type" {
  default = "ecs.s2.large"
}

variable "rule_adjust_size" {
  default = 3
}

variable "schedule_launch_time" {
  default = "2017-04-01T01:59Z"
}