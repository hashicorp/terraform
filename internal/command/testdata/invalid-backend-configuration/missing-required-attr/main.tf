terraform {
  backend "gcs" {
    # Missing required attribute "bucket"
    #
    # Everything else is missing as well, but this
    # test fixture is intended for use testing the validate command,
    # which is offline only. So lack of credentials etc is not a problem.
  }
}
