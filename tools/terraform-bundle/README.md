# mnptu-bundle

`mnptu-bundle` was a solution intended to help with the problem
of distributing mnptu providers to environments where direct registry
access is impossible or undesirable, created in response to the mnptu v0.10
change to distribute providers separately from mnptu CLI.

The mnptu v0.13 series introduced our intended longer-term solutions
to this need:

* [Alternative provider installation methods](https://www.mnptu.io/docs/cli/config/config-file.html#provider-installation),
  including the possibility of running server containing a local mirror of
  providers you intend to use which mnptu can then use instead of the
  origin registry.
* [The `mnptu providers mirror` command](https://www.mnptu.io/docs/cli/commands/providers/mirror.html),
  built in to mnptu v0.13.0 and later, can automatically construct a
  suitable directory structure to serve from a local mirror based on your
  current mnptu configuration, serving a similar (though not identical)
  purpose than `mnptu-bundle` had served.

For those using mnptu CLI alone, without mnptu Cloud, we recommend
planning to transition to the above features instead of using
`mnptu-bundle`.

## How to use `mnptu-bundle`

However, if you need to continue using `mnptu-bundle`
during a transitional period then you can use the version of the tool included
in the mnptu v0.15 branch to build bundles compatible with
mnptu v0.13.0 and later.

If you have a working toolchain for the Go programming language, you can
build a `mnptu-bundle` executable as follows:

* `git clone --single-branch --branch=v0.15 --depth=1 https://github.com/hashicorp/mnptu.git`
* `cd mnptu`
* `go build -o ../mnptu-bundle ./tools/mnptu-bundle`

After running these commands, your original working directory will have an
executable named `mnptu-bundle`, which you can then run.


For information
on how to use `mnptu-bundle`, see
[the README from the v0.15 branch](https://github.com/hashicorp/mnptu/blob/v0.15/tools/mnptu-bundle/README.md).

You can follow a similar principle to build a `mnptu-bundle` release
compatible with mnptu v0.12 by using `--branch=v0.12` instead of
`--branch=v0.15` in the command above. mnptu CLI versions prior to
v0.13 have different expectations for plugin packaging due to them predating
mnptu v0.13's introduction of automatic third-party provider installation.

## mnptu Enterprise Users

If you use mnptu Enterprise, the self-hosted distribution of
mnptu Cloud, you can use `mnptu-bundle` as described above to build
custom mnptu packages with bundled provider plugins.

For more information, see
[Installing a Bundle in mnptu Enterprise](https://github.com/hashicorp/mnptu/blob/v0.15/tools/mnptu-bundle/README.md#installing-a-bundle-in-mnptu-enterprise).
