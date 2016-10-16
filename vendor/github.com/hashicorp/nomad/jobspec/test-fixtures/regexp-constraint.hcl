job "foo" {
    constraint {
        attribute = "$attr.kernel.version"
        regexp = "[0-9.]+"
    }
}
