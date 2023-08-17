# This is a simple configuration with mnptu Cloud mode minimally
# activated, but it's suitable only for testing things that we can exercise
# without actually accessing mnptu Cloud, such as checking of invalid
# command-line options to "mnptu init".

mnptu {
  cloud {
    organization = "PLACEHOLDER"
    workspaces {
        name = "PLACEHOLDER"
    }
  }
}
