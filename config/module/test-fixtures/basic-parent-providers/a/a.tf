provider "top" {}

provider "bottom" {
    alias = "foo"
    value = "from bottom"
}

module "c" {
    source = "../c"
    providers = {
        "bottom" = "bottom.foo"
    }
}
