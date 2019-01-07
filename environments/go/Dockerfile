ARG GO_VERSION=1.9.7

FROM golang:${GO_VERSION} AS builder

ENV GOPATH /usr
ENV APP	   ${GOPATH}/src/github.com/fission/fission/environments/go

ADD context	    ${APP}/context
ADD server.go   ${APP}

WORKDIR ${APP}
RUN go get
RUN go build -a -o /server server.go

FROM zhangjyr/openfaas-ics:b1 AS ics

FROM ubuntu:18.04

RUN apt update && apt install -y ca-certificates && rm -rf /var/lib/apt/lists/*

WORKDIR /
COPY --from=builder /server /

RUN mkdir -p /tmp
COPY --from=ics /fwatchdog  /

ENV fprocess="/server -port %d"

HEALTHCHECK --interval=5s CMD [ -e /tmp/.lock ] || exit 1

CMD ["/fwatchdog"]

EXPOSE 8080
EXPOSE 8079
