module "example" {
  source = "./modules/example"
}

module "example2" {
  source = "./modules/${module.example.name}"
}
