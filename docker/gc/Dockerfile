FROM alpine:3.17

RUN apk add --update --no-cache ca-certificates git \
    && adduser -D -u 1000 jx

ENV JX_HOME /home/jx
USER 1000

COPY ./bin/gc-jobs /home/jx/
ENTRYPOINT ["/home/jx/gc-jobs"]
