
run "defaults" {
    command = plan

    assert {
        condition = tfcoremock_simple_resource.resource.string == "Hello, world!"
        error_message = "bad string value"
    }
}

run "overrides" {
    variables {
        input = "Hello, universe!"
    }

    assert {
        condition = tfcoremock_simple_resource.resource.string == "Hello, universe!"
        error_message = "bad string value"
    }
}
