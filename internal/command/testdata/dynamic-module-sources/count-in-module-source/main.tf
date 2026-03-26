module "example" {
  count  = 2
  source = "./modules/${count.index}"
}
