VERSION=0.1

.PHONY : test cover deps
test: 
	godep go test ./...
cover:
	./cover.sh
deps:
	go get github.com/tools/godep
	go get golang.org/x/tools/cmd/goimports
	go get github.com/mattn/goveralls
	godep restore
