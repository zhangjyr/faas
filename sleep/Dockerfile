FROM debian:jessie as builder
RUN apt-get update -q
RUN apt-get install -qy curl build-essential

RUN mkdir -p /sleep/

WORKDIR /sleep/
COPY sleep.c     .
COPY Makefile    .

RUN make sleep

FROM scratch

COPY --from=builder /sleep/bin/sleep /sleep
ENTRYPOINT ["/sleep"]
