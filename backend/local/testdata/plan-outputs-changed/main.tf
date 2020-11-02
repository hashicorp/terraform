module "submodule" {
  source = "./submodule"
}

output "changed" {
  value = "after"
}

output "sensitive_before" {
  value = "after"
  # no sensitive = true here, but the prior state is marked as sensitive in the test code
}

output "sensitive_after" {
  value = "after"

  # This one is _not_ sensitive in the prior state, but is transitioning to
  # being sensitive in our new plan.
  sensitive = true
}

output "added" { // not present in the prior state
  value = "after"
}

output "unchanged" {
  value = "before"
}
