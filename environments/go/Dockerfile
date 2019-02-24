ARG GO_VERSION=1.9.7

FROM golang:${GO_VERSION} AS builder

ENV GOPATH /usr
ENV APP	   ${GOPATH}/src/github.com/openfaas/faas/environments/go

ADD context	    ${APP}/context
ADD server.go   ${APP}
ADD example	    ${APP}/example
ADD vendor	    ${APP}/vendor

WORKDIR ${APP}
RUN go get
RUN go build -a -o /server server.go
RUN go build -buildmode=plugin -a -o /hello example/hello.go
RUN go build -buildmode=plugin -a -o /bye example/bye.go
RUN go build -buildmode=plugin -a -o /benchmark_matrix example/benchmark_matrix.go
RUN go build -buildmode=plugin -a -o /benchmark_cpu example/benchmark_cpu.go

FROM zhangjyr/hyperfaas-ics:b1 AS ics

FROM ubuntu:18.04

RUN apt update && apt install -y ca-certificates && rm -rf /var/lib/apt/lists/*

WORKDIR /
RUN mkdir -p /example
COPY --from=builder /server /
COPY --from=builder /hello /example/
COPY --from=builder /bye /example/
COPY --from=builder /benchmark_matrix /example/
COPY --from=builder /benchmark_cpu /example/

RUN mkdir -p /tmp
COPY --from=ics /ics  /

ENV fprocess="/server -port %d"
ENV faasBasePath="./example"

HEALTHCHECK --interval=5s CMD [ -e /tmp/.lock ] || exit 1

CMD ["/ics"]

EXPOSE 8080
EXPOSE 8079
