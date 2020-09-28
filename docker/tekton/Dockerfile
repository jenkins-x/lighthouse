FROM alpine:3.12

RUN apk add --update --no-cache ca-certificates git \
    && adduser -D jx

USER jx

COPY ./bin/lighthouse-tekton-controller /home/jx/
ENTRYPOINT ["/home/jx/lighthouse-tekton-controller"]
