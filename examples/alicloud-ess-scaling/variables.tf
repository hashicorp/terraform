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