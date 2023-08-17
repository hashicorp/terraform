mnptu {
    required_providers {
        mnptu = {
            // hashicorp/mnptu is published in the registry, but it is
            // archived (since it is internal) and returns a warning:
            //
            // "This provider is archived and no longer needed. The mnptu_remote_state
            // data source is built into the latest mnptu release."
            source = "hashicorp/mnptu"
        }
    }
}
