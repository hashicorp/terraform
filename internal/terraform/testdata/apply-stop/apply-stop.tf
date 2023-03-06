terraform {
  required_providers {
    indefinite = {
      source = "terraform.io/test/indefinite"
    }
  }
}

# The TestContext2Apply_stop test arranges for "indefinite"'s
# ApplyResourceChange to just block indefinitely until the operation
# is cancelled using Context.Stop.
resource "indefinite" "foo" {
}

resource "indefinite" "bar" {
  # Should never get here during apply because we're going to interrupt the
  # run during indefinite.foo's ApplyResourceChange.
  depends_on = [indefinite.foo]
}

output "result" {
  value = indefinite.foo.result
}
