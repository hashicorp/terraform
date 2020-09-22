module "source" {
  source = "./source"
}
module "child" {
  source = "./child"
  length = module.source.length
}
