run "test" {
    variables {
        input = "Hello, world!"
    }

    assert {
        condition = test_resource.resource.value == "Hello, world!"
        error_message = "wrong condition"
    }
}
