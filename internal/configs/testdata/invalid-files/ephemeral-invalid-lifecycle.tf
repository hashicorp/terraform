ephemeral "test_resource" "test" {
    lifecycle {
        create_before_destroy = true
    }
}