FROM alpine:3.12

RUN apk add --update --no-cache ca-certificates git \
    && adduser -D jx

ENV JX_HOME /home/jx
USER jx

COPY ./bin/gc-jobs /home/jx/
ENTRYPOINT ["/home/jx/gc-jobs"]
