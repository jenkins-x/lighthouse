FROM alpine:3.12

RUN apk add --update --no-cache ca-certificates git

COPY ./bin/keeper /keeper
ENTRYPOINT ["/keeper"]
