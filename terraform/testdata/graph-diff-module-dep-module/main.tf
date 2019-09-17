module "foo" {
    source = "./foo"
}

module "bar" {
    source = "./bar"
    in = "${module.foo.data}"
}
