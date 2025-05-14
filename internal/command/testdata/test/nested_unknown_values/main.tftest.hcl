
run "first" {

  command = plan

  variables {
    input = {
      one = "one"
      two = "two"
    }
  }
}

run "second" {

  command = plan

  variables {
    input = {
      # This should be okay, as run.first.one is unknown but we're not
      # referencing it directly.
      one = "one"
      two = run.first.two
    }
  }
}

run "third" {
  variables {
    # This should fail as one of the values in run.second is unknown.
    input = run.second
  }
}