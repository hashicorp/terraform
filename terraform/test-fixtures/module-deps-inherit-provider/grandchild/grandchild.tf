
# Here we *override* the foo from the parent
provider "foo" {

}

# We also use the "bar" provider defined at the root, which was
# completely ignored by the child module in between.
resource "bar_thing" "test" {

}
