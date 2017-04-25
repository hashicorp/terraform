## Contributing

The workflow is pretty standard:

1. Fork it
2. Create your feature branch (`git checkout -b my-new-feature`)
3. Commit your changes (`git commit -am 'Add some feature'`)
4. Push to the branch (`git push -u origin my-new-feature`)
5. Create new Pull Request

## Running Tests

First run `bin/ci/before_build.sh` that will create a vhost and user(s) needed
by the test suite.

The project uses [Ginkgo](http://onsi.github.io/ginkgo/) and [Gomega](https://github.com/onsi/gomega).

To clone dependencies and run tests, use `make`. It is also possible
to use the brilliant [Ginkgo CLI runner](http://onsi.github.io/ginkgo/#the-ginkgo-cli) e.g.
to only run a subset of tests.
