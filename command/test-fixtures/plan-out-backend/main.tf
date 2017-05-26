terraform {
    backend "http" {
        test = true
    }
}

resource "test_instance" "foo" {
    ami = "bar"
}
