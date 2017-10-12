provider "top" {}

provider "bottom" {
    alias = "foo"
    value = "from bottom"
}

module "b" {
    source = "../c"
    providers = {
        "bottom" = "bottom.foo"
    }
}
