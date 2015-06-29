FROM alpine:3.2
ENTRYPOINT ["/bin/connectable"]
COPY . /go/src/github.com/gliderlabs/connectable
RUN apk add --update go git mercurial iptables \
  && cd /go/src/github.com/gliderlabs/connectable \
  && export GOPATH=/go \
  && go get \
  && go build -ldflags "-X main.Version $(cat VERSION)" -o /bin/connectable \
  && apk del go git mercurial \
  && rm -rf /go /var/cache/apk/*
