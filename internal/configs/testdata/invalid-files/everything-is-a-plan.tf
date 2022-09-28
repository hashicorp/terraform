# experiments.EverythingIsAPlan exists but is not registered as an active (or
# concluded) experiment, so this should fail until the experiment "gate" is
# removed.
terraform {
  experiments = [everything_is_a_plan]
}

moved {
    from = test_instance.foo
    to   = test_instance.bar
}