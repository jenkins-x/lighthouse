FROM alpine:3.12

RUN apk add --update --no-cache ca-certificates git

RUN mkdir /jxhome
ENV JX_HOME /jxhome

COPY ./bin/foghorn /foghorn
ENTRYPOINT ["/foghorn"]
