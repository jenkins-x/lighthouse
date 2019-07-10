FROM golang:1.12.6

COPY . /go/src/github.com/jenkins-x/lighthouse
WORKDIR /go/src/github.com/jenkins-x/lighthouse
RUN make build-linux

FROM centos:7

RUN yum install -y git

COPY --from=0 /go/src/github.com/jenkins-x/lighthouse/bin/lighthouse /lighthouse

CMD ["/lighthouse"]