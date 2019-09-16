terraform {
    backend "foo" {
        foo = "bar"
        bar = ["baz"]
        map = { a = "b" }
    }
}
