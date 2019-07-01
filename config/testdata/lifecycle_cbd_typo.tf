resource "foo" "bar" {
    lifecycle {
        create_before_destroyy = false
    }
}
