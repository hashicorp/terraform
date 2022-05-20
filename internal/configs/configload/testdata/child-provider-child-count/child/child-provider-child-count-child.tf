provider "boop" {
    blah = true
}

module "grandchild" {
    source = "../grandchild"
}
