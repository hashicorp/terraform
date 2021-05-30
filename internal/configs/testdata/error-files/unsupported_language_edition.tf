terraform {
  # If a future change in this repository happens to make TF2038 a valid
  # edition then this will start failing; in that case, change this file to
  # select a different edition that isn't supported.
  language = TF2038 # ERROR: Unsupported language edition
}
