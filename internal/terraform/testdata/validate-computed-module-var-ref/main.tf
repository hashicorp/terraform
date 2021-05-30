module "source" {
  source = "./source"
}

module "dest" {
  source = "./dest"
  destin = "${module.source.sourceout}"
}
