FROM alpine:3.23

ARG ARCH=amd64
RUN apk add --update --no-cache ca-certificates git \
    && adduser -D -u 1000 jx

ENV JX_HOME /home/jx
USER 1000

COPY ./bin/foghorn-$ARCH /home/jx/foghorn
ENTRYPOINT ["/home/jx/foghorn"]
