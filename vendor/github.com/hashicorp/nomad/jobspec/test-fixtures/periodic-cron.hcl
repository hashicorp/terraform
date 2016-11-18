job "foo" {
    periodic {
        cron = "*/5 * * *"
        prohibit_overlap = true
    }
}
