FROM golang:1.12.6

ARG VERSION
ENV VERSION ${VERSION:-v0.0.0-preview}

COPY . /go/src/github.com/jenkins-x/lighthouse
WORKDIR /go/src/github.com/jenkins-x/lighthouse
RUN make test build-linux

FROM centos:7

RUN yum update  -y
RUN yum install -y epel-release && \
  yum install -y unzip \
  which \
  make \
  wget \
  zip \
  bzip2

# Git
ENV GIT_VERSION 2.21.0
RUN yum install -y curl-devel expat-devel gettext-devel openssl-devel zlib-devel && \
  yum install -y gcc perl-ExtUtils-MakeMaker && \
  cd /usr/src  && \
  wget https://www.kernel.org/pub/software/scm/git/git-${GIT_VERSION}.tar.gz  && \
  tar xzf git-2.21.0.tar.gz  && \
  cd git-2.21.0 && \
  make prefix=/usr/local/git all  && \
  make prefix=/usr/local/git install

ENV PATH /usr/local/git/bin:$PATH

COPY --from=0 /go/src/github.com/jenkins-x/lighthouse/bin/lighthouse /lighthouse

CMD ["/lighthouse"]