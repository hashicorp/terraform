variable "base_cidr_block" {
  type = string
}

variable "subnet_count" {
  type = number
}

locals {
  subnet_newbits = log(var.subnet_count, 2)
  subnet_cidr_blocks = toset([
    for n in range(var.subnet_count) : cidrsubnet(var.base_cidr_block, local.subnet_newbits, n)
  ])
}

resource "test_thing" "vpc" {
  string = var.base_cidr_block
}

resource "test_thing" "subnet" {
  for_each = local.subnet_cidr_blocks

  string = test_thing.vpc.string
  single {
    z = each.value
  }
}

resource "test_thing" "route_table" {
  for_each = local.subnet_cidr_blocks

  string = each.value
}

output "vpc_id" {
  value = test_thing.vpc.string
}

output "subnet_ids" {
  value = { for k, sn in test_thing.subnet : k => sn.string }
}
