
module "foo" {
  source = "./foo"
  # this block intentionally left (almost) blank
}

module "bar" {
  source = "hashicorp/bar/aws"

  boom = "🎆"
  yes  = true
}

module "baz" {
  source = "git::https://example.com/"

  a = 1

  count = 12
  for_each = ["a", "b", "c"]

  depends_on = [
    module.bar,
  ]
}

