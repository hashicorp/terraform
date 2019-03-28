# TF-UPGRADE-TODO: In Terraform v0.11 and earlier, it was possible to begin a
# resource name with a number, but it is no longer possible in Terraform v0.12.
#
# Rename the resource and run `terraform state mv` to apply the rename in the
# state. Detailed information on the `state mv` command can be found in the
# documentation online: https://www.terraform.io/docs/commands/state/mv.html
resource "test_instance" "1_invalid_resource_name" {
}
