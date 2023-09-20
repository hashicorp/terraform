
run "defaults" {
    command = plan

    assert {
        condition = output.input == "Hello, world!"
        error_message = "bad string value"
    }
}

run "overrides" {
    variables {
        input = "Hello, universe!"
    }

    assert {
        condition = output.input == "Hello, universe!"
        error_message = "bad string value"
    }
}
