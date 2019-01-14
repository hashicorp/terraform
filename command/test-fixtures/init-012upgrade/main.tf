resource "null_resource" "foo" {
  # This construct trips up the HCL2 parser because it looks like a nested block
  # but has quoted keys like a map. The upgrade tool would add an equals sign
  # here to turn this into a map attribute, but "terraform init" must first
  # be able to install the null provider so the upgrade tool can know that
  # "triggers" is a map attribute.
  triggers {
    "foo" = "bar"
  }
}
