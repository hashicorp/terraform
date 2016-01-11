FROM docker-dev.yelpcorp.com/lucid_yelp
MAINTAINER Tomas Doran <bobtfish@bobtfish.net>

RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -yq \
    go \
    git \
    build-essential \
    ruby1.9.1 rubygems1.9.1 \
    libopenssl-ruby1.9.1 \
    ruby1.9.1-dev \
    --no-install-recommends

ENV GOPATH /go
ENV RUBYOPT="-KU -E utf-8:utf-8"
RUN gem install fpm
RUN ln -s /var/lib/gems/1.9.1/bin/fpm /usr/local/bin/fpm

#WORKDIR /go/src/github.com/Yelp/terraform-provider-gitfile
