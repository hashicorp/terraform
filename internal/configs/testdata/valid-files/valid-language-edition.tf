terraform {
  # If we drop support for TF2021 in a future Terraform release then this
  # test will fail. In that case, update this to a newer edition that is
  # still supported, because the purpose of this test is to verify that
  # we can successfully decode the language argument, not specifically
  # that we support TF2021.
  language = TF2021
}
