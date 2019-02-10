FROM golang:1.9.7 as build
ARG VERSION
ARG GIT_COMMIT

RUN mkdir -p /go/src/github.com/openfaas/faas/ics
WORKDIR /go/src/github.com/openfaas/faas/ics

COPY main.go                .
COPY handler.go             .
COPY scheduler.go           .
COPY readconfig.go          .
COPY readconfig_test.go     .
COPY types                  types
COPY proxy                  proxy
COPY logger                 logger
COPY vendor                 vendor
COPY version.go             .

# Run a gofmt and exclude all vendored code.
# RUN test -z "$(gofmt -l $(find . -type f -name '*.go' -not -path "./vendor/*"))"

RUN go test -v ./...
# Stripping via -ldflags "-s -w"
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags "-s -w \
        -X main.GitCommit=$GIT_COMMIT \
        -X main.Version=$VERSION" \
        -installsuffix cgo -o ics . \
    && GOARM=7 GOARCH=arm CGO_ENABLED=0 GOOS=linux go build -a -ldflags "-s -w \
        -X main.GitCommit=$GIT_COMMIT \
        -X main.Version=$VERSION" \
        -installsuffix cgo -o ics-armhf . \
    && GOARCH=arm64 CGO_ENABLED=0 GOOS=linux go build -a -ldflags "-s -w \
        -X main.GitCommit=$GIT_COMMIT \
        -X main.Version=$VERSION" \
        -installsuffix cgo -o ics-arm64 . \
    && GOOS=windows CGO_ENABLED=0 go build -a -ldflags "-s -w \
        -X main.GitCommit=$GIT_COMMIT \
        -X main.Version=$VERSION" \
        -installsuffix cgo -o ics.exe .

FROM alpine:3.7

RUN mkdir -p /tmp
COPY --from=build /go/src/github.com/openfaas/faas/ics/ics  /ics

HEALTHCHECK --interval=5s CMD [ -e /tmp/.lock ] || exit 1

CMD ["/ics"]

EXPOSE 8080
EXPOSE 8079
