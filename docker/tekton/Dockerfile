FROM alpine:3.12

RUN apk add --update --no-cache ca-certificates git

COPY ./bin/lighthouse-tekton-controller /lighthouse-tekton-controller
ENTRYPOINT ["/lighthouse-tekton-controller"]
