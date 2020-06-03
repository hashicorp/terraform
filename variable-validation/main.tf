module "with_validations" {
  source = "./with-validations"

  # These are intentionally written to fail validation, to show what it looks
  # like when a validation rule fails.
  network_id = "image-12345"
  start_time = 123563456345
}
