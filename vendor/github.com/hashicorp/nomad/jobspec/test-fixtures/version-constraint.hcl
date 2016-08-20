job "foo" {
    constraint {
        attribute = "$attr.kernel.version"
        version = "~> 3.2"
    }
}
