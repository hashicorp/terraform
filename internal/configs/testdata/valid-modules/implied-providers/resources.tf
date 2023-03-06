// These resources map to the configured "foo" provider"
resource foo_resource "a" {}
data foo_resource "b" {}

// These resources map to a default "hashicorp/null" provider
resource null_resource "c" {}
data null_resource "d" {}

// These resources map to the configured "whatever" provider, which has FQN
// "acme/something".
resource whatever_resource "e" {}
data whatever_resource "f" {}
