# There is no provider in required_providers called "grandchild", so this
# implicitly declares a dependency on "hashicorp/grandchild".
resource "grandchild_foo" "bar" {
}
