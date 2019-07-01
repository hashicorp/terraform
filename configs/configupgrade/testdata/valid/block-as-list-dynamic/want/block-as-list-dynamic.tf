resource "test_instance" "foo" {
  dynamic "network" {
    for_each = var.baz
    content {
      # TF-UPGRADE-TODO: The automatic upgrade tool can't predict
      # which keys might be set in maps assigned here, so it has
      # produced a comprehensive set here. Consider simplifying
      # this after confirming which keys can be set in practice.

      cidr_block = lookup(network.value, "cidr_block", null)

      dynamic "subnet" {
        for_each = lookup(network.value, "subnet", [])
        content {
          number = subnet.value.number
        }
      }
    }
  }
}
