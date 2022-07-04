FROM alpine:3.16

RUN apk add --update --no-cache ca-certificates git \
    && adduser -D -u 1000 jx

USER 1000

COPY ./bin/lighthouse-tekton-controller /home/jx/
ENTRYPOINT ["/home/jx/lighthouse-tekton-controller"]
