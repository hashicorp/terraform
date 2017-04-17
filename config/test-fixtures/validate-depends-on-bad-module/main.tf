module "child" {
    source = "./child"
}

resource aws_instance "web" {
    depends_on = ["module.foo"]
}
