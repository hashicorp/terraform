terraform {
    backend "local-no-default" {
        workspace_dir = "envdir-new"
    }
}
