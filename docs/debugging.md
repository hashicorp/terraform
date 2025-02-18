# How to Debug Terraform

Contents:
- [Debugging automated tests](#debugging-automated-tests)
    - [Debugging automated tests in VSCode](#debugging-automated-tests-in-vscode)

As Terraform is written in Go you may use [Delve](https://github.com/go-delve/delve) to debug it.

GoLand includes [debugging features](https://www.jetbrains.com/help/go/debugging-code.html), and the [Go extension for VS Code](https://code.visualstudio.com/docs/languages/go#_debugging) makes it easy to use Delve when debugging Go codebases in VS Code. 

## Debugging automated tests

Debugging an automated test is often the most straightforward workflow for debugging a section of the codebase. For example, the Go extension for VS Code](https://code.visualstudio.com/docs/languages/go#_debugging) adds `run test | debug test` options above all tests in a `*_test.go` file. These allow debugging without any prior configuration.

### Debugging automated tests in VSCode

As described above, debugging tests in VS Code is easily achieved through the Go extension.

If you need more control over how tests are run while debugging, e.g. environment variable values, look at the [example debugger launch configuration 'Run selected test'](./debugging-configs/vscode/debug-automated-tests/launch.json). You can adapt this example to create your own [launch configuration file](https://code.visualstudio.com/docs/editor/debugging#_launch-configurations).

When using this debugger configuration you must highlight a test's name and launch the debugger configuration:

<p align="center">
    <img width="75%" alt="Debugging a single test using the example 'Run selected test' debugger configuration shared in this repository" src="./images/vscode-debugging-test.png"/>
</p>

## 1. Compile & Start Debug Server

One way to do it is to compile a binary with the [appropriate compiler flags](https://pkg.go.dev/cmd/compile#hdr-Command_Line):

```sh
go install -gcflags="all=-N -l"
```

This enables you to then execute the compiled binary via Delve, pass any arguments and spin up a debug server which you can then connect to:

```sh
dlv exec $GOBIN/terraform --headless --listen :2345 --log -- apply
```

## 2a. Connect via CLI

You may connect to the headless debug server via Delve CLI

```sh
dlv connect :2345
```

## 2b. Connect from VS Code

The repository provides a launch configuration, making it possible to use VS Code's native debugging integration:

![vscode debugger](./images/vscode-debugging.png)

Note that this debugging workflow is different from the test-based one, which itself shouldn't require any of the above steps above nor the mentioned launch configuration. Meaning, that if you already have a test that hits the right lines of code you want to be debugging, or you can write one, then that may be an easier workflow.
