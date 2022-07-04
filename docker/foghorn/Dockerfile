FROM alpine:3.16

RUN apk add --update --no-cache ca-certificates git \
    && adduser -D -u 1000 jx

ENV JX_HOME /home/jx
USER 1000

COPY ./bin/foghorn /home/jx/
ENTRYPOINT ["/home/jx/foghorn"]
