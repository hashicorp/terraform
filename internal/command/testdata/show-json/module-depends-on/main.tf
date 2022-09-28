module "foo" {
    source = "./foo"

    depends_on = [
        test_instance.test
    ]
}

resource "test_instance" "test" {
    ami   = "foo-bar"
}
