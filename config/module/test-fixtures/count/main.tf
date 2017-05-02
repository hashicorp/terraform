# Hello

module "foo" {
  source = "./foo"
}

module "foofoo" {
  source = "./foo"
  count  = "2"
}
