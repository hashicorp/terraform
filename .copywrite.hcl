schema_version = 1

project {
  license        = "BUSL-1.1"
  copyright_year = 2024

  # (OPTIONAL) A list of globs that should not have copyright/license headers.
  # Supports doublestar glob patterns for more flexibility in defining which
  # files or folders should be ignored
  header_ignore = [
    "**/*.tf",
    "**/testdata/**",
    "**/*.pb.go",
    "**/*_string.go",
    "**/mock*.go",
    ".changes/**",
    # these directories have their own copywrite config
    "docs/plugin-protocol/**",
    "internal/tfplugin*/**"
  ]
}
