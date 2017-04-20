FROM golang:1.6.2

RUN go get  github.com/golang/lint/golint \
            github.com/mattn/goveralls \
            golang.org/x/tools/cover \
            github.com/tools/godep \
            github.com/aktau/github-release

ENV USER root
WORKDIR /go/src/github.com/HewlettPackard/oneview-golang

COPY . /go/src/github.com/HewlettPackard/oneview-golang
RUN mkdir bin
