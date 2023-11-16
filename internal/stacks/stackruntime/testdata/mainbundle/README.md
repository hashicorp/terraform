# Synthetic source bundle for most tests

Since the tests in this package are concerned primilary with configuration
evaluation and less concerned about configuration bundling or loading,
most of our tests can just use subdirectories of the only package in this
synthetic source bundle to avoid the inconvenience of maintaining an entire
source bundle for each separate test.

To use this:
- Make a subdirectory under `test/` with a name that's related to your test
  case(s).
- Use the `loadMainBundleConfigForTest` helper, passing the name of your
  test directory as the source directory.

    (The helper function will automatically construct the synthetic remote
    source address needed to locate that subdirectory within the source bundle.)
