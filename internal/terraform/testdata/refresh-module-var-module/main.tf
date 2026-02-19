module "foo" {
    source = "./foo"
}

module "bar" {
    source = "./bar"
    value = "${module.foo.output}"
}
