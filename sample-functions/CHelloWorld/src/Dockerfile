FROM debian:jessie as builder
RUN apt-get update -q
RUN apt-get install -qy curl build-essential

WORKDIR /root/

COPY main.c     .

RUN gcc main.c -static -o /main \
 && chmod +x /main \
 && /main

FROM zhangjyr/sleep:0.2

COPY --from=builder /main               /
