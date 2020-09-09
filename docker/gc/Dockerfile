FROM alpine:3.10

RUN apk add --update --no-cache ca-certificates git

RUN mkdir /jxhome
ENV JX_HOME /jxhome

COPY ./bin/gc-jobs /gc-jobs
ENTRYPOINT ["/gc-jobs"]
