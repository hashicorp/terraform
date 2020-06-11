resource "foo_resource" "b" {}
resource "bar_resource" "c" {}
resource "bar_resource" "ab" {
  provider = baz
}
resource "terraform_remote_state" "production" {}
