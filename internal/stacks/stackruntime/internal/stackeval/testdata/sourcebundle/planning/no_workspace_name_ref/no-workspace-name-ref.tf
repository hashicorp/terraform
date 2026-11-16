output "invalid" {
  # terraform.workspace is not available when this module is used as part
  # of a stack component, so this should produce an error during
  # planning.
  value = terraform.workspace
}
