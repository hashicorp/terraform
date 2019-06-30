module "foo" {
    source = "./foo"
    foo = "${self.bar}"
}
