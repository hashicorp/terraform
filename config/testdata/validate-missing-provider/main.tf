provider "test" {
    alias = "bar"
}

module "mod" {
    source = "./mod"
    providers = {
        "test" = "test.foo"
    }
}
