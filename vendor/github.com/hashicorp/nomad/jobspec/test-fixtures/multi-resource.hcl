job "binstore-storagelocker" {
    group "binsl" {
        count = 5
        task "binstore" {
            driver = "docker"

            resources {
                cpu = 500
                memory = 128
            }

            resources {
                cpu = 500
                memory = 128
            }
        }
    }
}
