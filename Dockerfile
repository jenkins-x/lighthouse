ARG GO_VERSION=1.12.7
ARG VERSION

FROM golang:${GO_VERSION} AS builder
ENV VERSION ${VERSION:-v0.0.0-preview}
COPY . /go/src/github.com/jenkins-x/lighthouse
WORKDIR /go/src/github.com/jenkins-x/lighthouse
RUN make build-linux

FROM alpine:3.10
RUN apk add --update --no-cache ca-certificates git 
COPY --from=builder /go/src/github.com/jenkins-x/lighthouse/bin/lighthouse /lighthouse
RUN mkdir /jxhome
ENV JX_HOME /jxhome
ENTRYPOINT ["/lighthouse"]
