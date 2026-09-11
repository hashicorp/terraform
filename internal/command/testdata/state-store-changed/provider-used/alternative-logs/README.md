The configuration in `internal/command/testdata/state-store-changed/provider-used` can be used to produce different types of logs depending on how the providers are downloaded. For example if the providers are sourced from a global cache then logs will mention that, instead of downloading from the Public Registry.

Currently, the logs alongside the configuration assumes that mock providers are supplied in the normal way, which mimics the experience of
using a filesystem mirror. The logs in the subfolder `/alternative-logs/automatic-provider-approval` instead describe the logs when the provider is downloaded via HTTP, which triggers some security features for establishing trust on first use.

