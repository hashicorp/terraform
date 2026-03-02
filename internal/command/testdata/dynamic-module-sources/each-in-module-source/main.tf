module "example" {
  for_each = toset(["one", "two"])
  source   = "./modules/${each.key}"
}
