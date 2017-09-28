#terraform:hcl2

locals {
  # This direct expression is something that would be rejected by the old HCL
  # parser, so we can use it as a marker that the HCL2 parser was used.
  foo = 1 + 2
}
