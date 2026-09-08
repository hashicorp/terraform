terraform {
  # Only the root module can declare a Cloud configuration. Terraform should emit a warning
  # about this child module Cloud declaration.
  cloud {
  }
}
