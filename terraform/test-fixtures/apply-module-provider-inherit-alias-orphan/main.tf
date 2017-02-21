provider "aws" {
    root = "1"
}

provider "aws" {
    child = "eu"
    alias = "eu"
}
