locals {
  environments = toset(["STAGE", "PROD"])
}

upstream "admin" {
  remote = "example.com/infrastructure/admin"
}

workspace "network" {
  for_each = local.environments

  config = "./network"
  variables = {
    admin_role_arn = upstream.admin.admin_role_arn
  }
}

locals {
  aws_subnet_ids = flatten(workspace.network.aws_vpcs[*].subnet_ids)
}

workspace "monitoring" {
  for_each = local.environments

  config = "./monitoring"
  variables = {
    admin_role_arn = upstream.admin.admin_role_arn
    aws_subnet_ids = local.aws_subnet_ids
  }
}

workspace "dns" {
  for_each = local.environments

  config = "./dns"
  variables = {
    domain = "${lower(each.key)}.example.com"
    records = concat(
      workspace.network.dns_records,
      workspace.monitoring.dns_records,
    )
  }
}
