terraform {
  backend "s3" {
    // No config, as this fixture is intended for tests using -backend=false,
    // so the fact this is completely unconfigured should not cause an error.
  }
}
