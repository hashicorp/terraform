terraform {
  required_providers {
    a = {
      # This one is just not available at all
      source = "example.com/test/a"
    }
    b = {
      # This one is unavailable but happens to be cached in the legacy
      # cache directory, under .terraform/plugins
      source = "example.com/test/b"
    }
    c = {
      # This one is also cached in the legacy cache directory, but it's
      # an official provider so init will assume it got there via normal
      # automatic installation and not generate a warning about it.
      # This one is also not available at all, but it's an official
      # provider so we don't expect to see a warning about it.
      source = "hashicorp/c"
    }
  }
}
