module "source" {
  source = "./source"
}
module "child" {
  source  = "./child"
  list = "${module.source.list}"
}
