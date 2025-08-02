module "example" {
  source = ""  # ERROR: Invalid module source address
  hashes = [
    "h1:test-hash",
  ]
}