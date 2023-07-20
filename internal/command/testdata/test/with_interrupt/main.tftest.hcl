variables {
  interrupts = 0
}

run "primary" {

}

run "secondary" {
  variables {
    interrupts = 1
  }
}

run "tertiary" {

}
