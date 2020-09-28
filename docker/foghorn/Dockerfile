FROM alpine:3.12

RUN apk add --update --no-cache ca-certificates git \
    && adduser -D jx

ENV JX_HOME /home/jx
USER jx

COPY ./bin/foghorn /home/jx/
ENTRYPOINT ["/home/jx/foghorn"]
