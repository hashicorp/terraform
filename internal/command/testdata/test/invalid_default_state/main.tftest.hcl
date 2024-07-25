run "test" {
    assert {
        condition = test_resource.resource.value == "Hello, world!"
        error_message = "wrong condition"
    }
}
