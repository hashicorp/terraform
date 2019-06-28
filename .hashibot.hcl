behavior "regexp_issue_labeler" "panic_label" {
    regexp = "panic:"
    labels = ["crash", "bug"]
}

behavior "regexp_issue_notifier" "panic_notify" {
    regexp = "panic:"
    slack_channel = env.TERRAFORM_SLACK_CHANNEL
    message = "Panic report! https://github.com/${var.repository}/issues/${var.issue_number} has a panic in it."
}

behavior "remove_labels_on_reply" "remove_stale" {
    labels = ["waiting-response", "stale"]
    only_non_maintainers = true
}

poll "label_issue_migrater" "provider_migrater" {
    schedule = "0 50 11 * * *"
    new_owner = env.PROVIDERS_OWNER
    repo_prefix = "terraform-provider-"
    label_prefix = "provider/"
    excluded_label_prefixes = ["backend/", "provisioner/"]
    excluded_labels = ["build", "cli", "config", "core", "new-provider", "new-provisioner", "new-remote-state", "provider/terraform"]
    aliases = {
        "provider/google-cloud" = "provider/google"
        "provider/influx" = "provider/influxdb"
        "provider/vcloud" = "provider/vcd"
    }
    issue_header = <<-EOF
    _This issue was originally opened by ${var.user} as ${var.repository}#${var.issue_number}. It was migrated here as a result of the [provider split](https://www.hashicorp.com/blog/upcoming-provider-changes-in-terraform-0-10/). The original body of the issue is below._
    
    <hr>
    
    EOF
    migrated_comment = "This issue has been automatically migrated to ${var.repository}#${var.issue_number} because it looks like an issue with that provider. If you believe this is _not_ an issue with the provider, please reply to ${var.repository}#${var.issue_number}."
}
