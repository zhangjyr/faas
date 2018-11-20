FROM debian:jessie as builder
RUN apt-get update -q
RUN apt-get install -qy curl build-essential

RUN curl -SL https://github.com/openfaas/faas/releases/download/0.9.6/fwatchdog > /usr/bin/fwatchdog \
    && chmod +x /usr/bin/fwatchdog

WORKDIR /root/

COPY main.c     .

RUN gcc main.c -static -o /main \
 && chmod +x /main \
 && /main

ENV VERSION 2.28
RUN curl -SL https://www.kernel.org/pub/linux/utils/util-linux/v$VERSION/util-linux-$VERSION.tar.gz | tar -zxf- \
    && ln -s util-linux-$VERSION util-linux \
    && cd /root/util-linux \
    && ./configure --without-ncurses

WORKDIR /root/util-linux
COPY nsenter.c ./sys-utils/nsenter.c
RUN make LDFLAGS=-all-static nsenter \
    && mv nsenter /

FROM alpine:3.7

COPY --from=builder /main               /
COPY --from=builder /usr/bin/fwatchdog  /
COPY --from=builder /nsenter            /
# COPY --from=builder /tmp /tmp

ENV fprocess="/nsenter -t 1 /main"
# ENV fprocess="/main"

HEALTHCHECK --interval=5s CMD [ -e /tmp/.lock ] || exit 1

CMD ["/fwatchdog"]

EXPOSE 8080
