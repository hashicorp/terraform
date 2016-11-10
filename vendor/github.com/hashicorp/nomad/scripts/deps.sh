#!/usr/bin/env bash

# First get the OS-specific packages
GOOS=windows go get $DEP_ARGS github.com/StackExchange/wmi
GOOS=windows go get $DEP_ARGS github.com/shirou/w32
GOOS=linux go get $DEP_ARGS github.com/docker/docker/pkg/mount
GOOS=linux go get $DEP_ARGS github.com/opencontainers/runc/libcontainer/cgroups/fs
GOOS=linux go get $DEP_ARGS github.com/opencontainers/runc/libcontainer/configs
GOOS=linux go get $DEP_ARGS github.com/coreos/go-systemd/util
GOOS=linux go get $DEP_ARGS github.com/coreos/go-systemd/dbus

# Get the rest of the deps
DEPS=$(go list -f '{{range .TestImports}}{{.}} {{end}}' ./...)
go get $DEP_ARGS ./... $DEPS
