# The global cache is only an eligible installation source if there's already
# a lock entry for the given provider and it contains at least one checksum
# that matches the cache entry.
#
# This lock file therefore matches the "not a real provider" fake executable
# under the "cache" directory, rather than the real provider from upstream,
# so that Terraform CLI will consider the cache entry as valid.

provider "registry.terraform.io/hashicorp/template" {
  version = "2.1.0"
  hashes = [
    "h1:e7YvVlRZlaZJ8ED5KnH0dAg0kPL0nAU7eEoCAZ/sOos=",
  ]
}
