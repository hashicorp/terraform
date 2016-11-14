job "foo" {
  task "bar" {
    driver = "docker"

    config {
      image = "hashicorp/image"

      port_map {
        db = 1234
      }
    }
  }
}
