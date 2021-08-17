boundary {} # ERROR: At least one connection must be defined in a Boundary block

boundary {
  connection "test" {}
  connection "test" {} # ERROR: Connection "test" has been defined multiple times
}

boundary {
  connection "hello" {}
}

boundary { # ERROR: Connection "hello" has been defined multiple times
  connection "hello" {}
}
