
# "foo" is inherited from the parent module
resource "foo_bar" "test" {

}

# but we don't use the "bar" provider inherited from the parent

# "baz" is introduced here for the first time, so it's an implicit
# dependency
resource "baz_bar" "test" {

}

module "grandchild" {
  source = "../grandchild"
}
