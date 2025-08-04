action "test_action" "test" {
    foo = "baz"
}

resource "test_instance" "test" {
    lifecycle {
      action_trigger {
        events = [after_destroy]
        actions = [action.test_action.dosomething]
      }
    }
}