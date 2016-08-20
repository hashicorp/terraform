job "binstore-storagelocker" {
  group "binsl" {
    count = 5

    task "binstore" {
      driver = "docker"

      config {
        image = ""
      }

      resources {
      }
    }
  }
}
