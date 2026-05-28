FROM alpine:3.23

ARG ARCH=amd64
RUN apk add --update --no-cache ca-certificates git \
    && adduser -D -u 1000 jx

USER 1000

COPY ./bin/keeper-$ARCH /home/jx/keeper
ENTRYPOINT ["/home/jx/keeper"]
