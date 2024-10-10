# How to Debug Terraform

As Terraform is written in Go you may use [delve](https://github.com/go-delve/delve) to debug it.

## 1. Compile & Start Debug Server

One way to do it is to compile a binary with the [appropriate compiler flags](https://pkg.go.dev/cmd/compile#hdr-Command_Line):

```sh
go install -gcflags="all=-N -l"
```

This enables you to then execute the compiled binary via delve, pass any arguments as spin up a debug server which you can then connect to:

```sh
dlv exec $GOBIN/terraform --headless --listen :2345 --log -- apply
```

## 2a. Connect via CLI

You may connect to the headless debug server via delve CLI

```sh
dlv connect :2345
```

## 2b. Connect from VS Code

The repository provides a launch configuration, making it possible to use VS Code's native debugging integration:

![vscode debugger](./images/vscode-debugging.png)

Note that this debugging workflow is different from the test-based one, which itself shouldn't require any of the above steps above nor the mentioned launch configuration. Meaning, that if you already have a test that hits the right lines of code you want to be debugging, or you can write one, then that may be an easier workflow.
