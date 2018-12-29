
# The use of an equals to assign "locals" is something that would be rejected
# by the HCL2 parser (equals is reserved for attributes only) and so we can
# use it to verify that the old HCL parser was used.
locals {
  foo = "bar"
}
