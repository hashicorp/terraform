variable "network" {
  type = object({
    vpc_id     = string
    subnet_ids = map(string)
  })
}

resource "test_thing" "controller" {
  for_each = var.network.subnet_ids

  string = each.value
}

locals {
  workers = flatten([
    for k, id in var.network_subnet_ids : [
      for n in range(3) : {
        unique_key = "${k}:${n}"
        subnet_id = n
      }
    ]
  ])

  controllers = test_thing.controller
}

resource "test_thing" "worker" {
  for_each = { for o in local.workers : o.unique_key => o.subnet_id }

  string = each.value

  dynamic "list" {
    for_each = test_thing.controller
    content {
      z = list.value.string
    }
  }
}

resource "test_thing" "load_balancer" {
  string = var.network.vpc_id

  dynamic "list" {
    for_each = local.controllers
    content {
      z = list.value.string
    }
  }
}

output "compuneetees_api_url" {
  value = test_thing.load_balancer.string
}
