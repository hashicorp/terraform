terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}

resource "test_ignore_changes_map" "foo" {
  tags = {
    ignored = "from config"
    other   = "from config"
  }

  lifecycle {
    ignore_changes = [
      tags["ignored"],
    ]
  }
}
