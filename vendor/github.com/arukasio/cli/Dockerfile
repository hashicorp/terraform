FROM arukasio/arukas:dev
MAINTAINER "Shuji Yamada <s-yamada@arukas.io>"

ENV REPO_ROOT $GOPATH/src/github.com/arukasio/cli

COPY . $REPO_ROOT
WORKDIR $REPO_ROOT

RUN godep restore
RUN for package in $(go list ./...| grep -v vendor); do golint ${package}; done
RUN ARUKAS_DEV=1 scripts/build.sh

WORKDIR $GOPATH

ENTRYPOINT ["bin/arukas"]
