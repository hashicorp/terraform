module "b" {
  source = "./modb"
}

module "a" {
  source = "./moda"

  single_element = "${element(module.b.computed_list, 0)}"
}
