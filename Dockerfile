FROM gcr.io/jenkinsxio/builder-go:2.0.1245-582
#RUN apk add --update --no-cache ca-certificates git
COPY ./bin/lighthouse  /lighthouse
RUN mkdir /jxhome
ENV JX_HOME /jxhome
ENTRYPOINT ["/lighthouse"]
