provider "foo" {
    value = "from root"
}

module "child" {
    source = "./child"
}
