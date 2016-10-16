job "binstore-storagelocker" {
    group "binsl" {
        task "binstore" {
            driver = "docker"

            artifact {
                source = "http://foo.com/bar"
                destination = ""
                options {
                    foo = "bar"
                }
            }

            artifact {
                source = "http://foo.com/baz"
            }
            artifact {
                source = "http://foo.com/bam"
                destination = "var/foo"
            }
            resources {}
        }
    }
}
