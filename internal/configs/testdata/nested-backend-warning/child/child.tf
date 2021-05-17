terraform {
  # Only the root module can declare a backend. Terraform should emit a warning
  # about this child module backend declaration.
  backend "ignored" {
  }
}
