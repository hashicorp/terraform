
module "a" {
  count  = 1
  source = "./a"
}

module "b" {
  count  = 1
  source = "./b"
}

resource "foo" "bar" {}
