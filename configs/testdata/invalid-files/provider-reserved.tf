provider "test" {
  # These are okay
  alias   = "foo"
  version = "1.0.0"

  # Provider-specific arguments are also okay
  arbitrary = true

  # These are all reserved and should generate errors.
  count      = 3
  depends_on = ["foo.bar"]
  for_each   = ["a", "b"]
  source     = "foo.example.com/baz/bar"
  lifecycle {}
  locals {}
}
