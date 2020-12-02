# This is for a specific hclwrite oddity with how it was handling attributes
# whose expressions are entirely enclosed in parentheses. This has now been
# fixed upstream and so this test is just to prove that and to guard against
# regressions.
#   https://github.com/hashicorp/terraform/issues/27040
locals {
  a = (
    1 + 2
  )
  b = local.a
}
