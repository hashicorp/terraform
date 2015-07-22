FROM golang:1.4.2
MAINTAINER Panagiotis Moustafellos <pmoust@peopleperhour.com>
ADD . /go/src/github.com/hashicorp/terraform
RUN mkdir /tf && \
    cd /go/src/github.com/hashicorp/terraform && \
    make updatedeps && \
    make dev 
WORKDIR "/tf"
VOLUME ["/tf"]
ENTRYPOINT ["$GOPATH/bin/terraform"]
