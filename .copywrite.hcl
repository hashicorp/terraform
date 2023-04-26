schema_version = 1

project {
  license        = "MPL-2.0"
  copyright_year = 2014

  # The HashiCorp CLA grants an irrevocable license to use community
  # contributions but does not include full copyright assignment.
  copyright_holder = "Hashicorp, Inc. and community contributors"

  header_ignore = [
    "**/*.tf",
    "**/testdata/**",
    "**/*.pb.go",
    "**/*_string.go",
    "**/mock*.go",
  ]
}
