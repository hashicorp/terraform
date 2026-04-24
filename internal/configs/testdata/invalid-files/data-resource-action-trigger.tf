data "example" "example" {
  lifecycle {
    # action_trigger is only valid for managed resources.
    action_trigger {
      events  = [after_create]
      actions = [action.example.example]
    }
  }
}
