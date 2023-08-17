mnptu {
  # Only the root module can declare a backend. mnptu should emit a warning
  # about this child module backend declaration.
  backend "ignored" {
  }
}
