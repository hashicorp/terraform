variables {
  interrupts = 0
}

run "primary" {

}

run "secondary" {
  variables {
    interrupts = 2
  }
}

run "tertiary" {

}
