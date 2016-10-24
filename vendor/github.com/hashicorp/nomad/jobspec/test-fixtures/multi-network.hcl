job "binstore-storagelocker" {
    group "binsl" {
        count = 5
        task "binstore" {
            driver = "docker"

            resources {
                cpu = 500
                memory = 128

                network {
                    mbits = "100"
                    reserved_ports = [1,2,3]
                    dynamic_ports = ["HTTP", "HTTPS", "ADMIN"]
                }

                network {
                    mbits = "128"
                    reserved_ports = [1,2,3]
                    dynamic_ports = ["HTTP", "HTTPS", "ADMIN"]
                }
            }
        }
    }
}
