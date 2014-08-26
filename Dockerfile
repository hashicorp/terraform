FROM ubuntu:14.04

MAINTAINER Steven Borrelli <steve@borrelli.org>

ENV DEBIAN_FRONTEND noninteractive 

RUN apt-get update

RUN apt-get -y install build-essential cmake git golang-go make mercurial

ENV CGOENABLED 1
ENV GOPATH /opt 
ENV BUILDDIR $GOPATH/src/github.com/hashicorp/terraform
ENV PATH $PATH:$GOPATH/bin

#Install Gox
RUN go get -u github.com/mitchellh/gox

ADD . $BUILDDIR


RUN cd $BUILDDIR && make updatedeps
RUN cd $BUILDDIR && make dev

ENTRYPOINT ["/opt/bin/terraform"]

