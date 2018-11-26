FROM debian:jessie as builder
RUN apt-get update -q
RUN apt-get install -qy curl build-essential

RUN mkdir -p /standby/

WORKDIR /standby/
COPY standby.c   .
COPY Makefile    .

RUN make standby

# FROM scratch
FROM alpine:3.7

COPY --from=builder /standby/bin/standby /standby
ENTRYPOINT ["/standby"]
