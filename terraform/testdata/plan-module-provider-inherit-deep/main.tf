module "A" {
    source = "./A"
}

provider "aws" {
    from = "root"
}
