provider "top" {
    alias = "foo"
    value = "from top"
}

module "a" {
    source = "./a"
    providers = {
        "top" = "top.foo"
    }
}
