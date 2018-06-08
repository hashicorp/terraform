#!/bin/bash
cd "$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
go get github.com/ChrisTrenkamp/goxpath/cmd/goxpath
if [ $? != 0 ]; then
	exit 1
fi
go test >/dev/null 2>&1
if [ $? != 0 ]; then
	go test
	exit 1
fi
gometalinter --deadline=1m ./...
go list -f '{{if gt (len .TestGoFiles) 0}}"go test -covermode count -coverprofile {{.Name}}.coverprofile -coverpkg ./... {{.ImportPath}}"{{end}} >/dev/null' ./... | xargs -I {} bash -c {} 2>/dev/null
gocovmerge `ls *.coverprofile` > coverage.txt
go tool cover -html=coverage.txt -o coverage.html
firefox coverage.html
rm coverage.html coverage.txt *.coverprofile
