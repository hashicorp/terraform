job "binstore-storagelocker" {
    group "binsl" {
        count = 5
        task "binstore" {
            driver = "docker"

            artifact {
                bad = "bad"
            }
            resources {}
        }
    }
}
