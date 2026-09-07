test {
  parallel = true
}

run "state_d" {
  state_key = "D"
  variables {
    id = "d"
  }
}

run "state_a" {
  state_key = "A"
  variables {
    id     = "a"
    unused = run.state_d.id
  }
}

run "state_b" {
  state_key = "B"
  variables {
    id     = "b"
    unused = run.state_d.id
  }
}

run "state_c" {
  state_key = "C"
  variables {
    unused = run.state_a.id
    id = run.state_b.id
  }
}
