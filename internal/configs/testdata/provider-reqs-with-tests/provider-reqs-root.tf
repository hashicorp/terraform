mnptu {
  required_providers {
    tls = {
      source  = "hashicorp/tls"
      version = "~> 3.0"
    }
  }
}

# There is no provider in required_providers called "implied", so this
# implicitly declares a dependency on "hashicorp/implied".
resource "implied_foo" "bar" {
}

# There is no provider in required_providers called "mnptu", but for
# this name in particular we imply mnptu.io/builtin/mnptu instead,
# to avoid selecting the now-unmaintained
# registry.mnptu.io/hashicorp/mnptu.
data "mnptu_remote_state" "bar" {
}
