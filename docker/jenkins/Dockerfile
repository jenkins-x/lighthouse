FROM alpine:3.12

RUN apk add --update --no-cache ca-certificates git

COPY ./bin/jenkins-controller /jenkins-controller
ENTRYPOINT ["/jenkins-controller"]
