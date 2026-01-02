FROM alpine:3.17

ARG ARCH=amd64
RUN apk add --update --no-cache ca-certificates git \
    && adduser -D -u 1000 jx

USER 1000

COPY ./bin/jenkins-controller-$ARCH /home/jx/jenkins-controller
ENTRYPOINT ["/home/jx/jenkins-controller"]
