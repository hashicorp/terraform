locals {
  environments = toset(["STAGE", "PROD"])
}

workspace "admin" {
}

workspace "network" {
  for_each = local.environments
}

workspace "monitoring" {
  for_each = local.environments
}
