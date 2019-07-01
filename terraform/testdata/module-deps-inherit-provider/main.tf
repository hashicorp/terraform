
provider "foo" {
}

provider "bar" {

}

module "child" {
    source = "./child"
}
