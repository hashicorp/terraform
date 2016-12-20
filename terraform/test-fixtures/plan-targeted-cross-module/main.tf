module "A" {
    source = "./A"
}

module "B" {
    source = "./B"
    input  = "${module.A.value}"
}
