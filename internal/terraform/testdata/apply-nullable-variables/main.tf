module "mod" {
  source = "./mod"
  nullable_null_default = null
  nullable_non_null_default = null
  nullable_no_default = null
  non_nullable_default = null
  non_nullable_no_default = "ok"
}

output "nullable_null_default" {
  value = module.mod.nullable_null_default
}

output "nullable_non_null_default" {
  value = module.mod.nullable_non_null_default
}

output "nullable_no_default" {
  value = module.mod.nullable_no_default
}

output "non_nullable_default" {
  value = module.mod.non_nullable_default
}

output "non_nullable_no_default" {
  value = module.mod.non_nullable_no_default
}
