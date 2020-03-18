FROM alpine:3.10
RUN apk add --update --no-cache ca-certificates git
COPY ./bin/lighthouse  /lighthouse
RUN mkdir /jxhome
ENV JX_HOME /jxhome
ENTRYPOINT ["/lighthouse"]
