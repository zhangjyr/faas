FROM debian:jessie as builder
RUN apt-get update -q
RUN apt-get install -qy curl build-essential

WORKDIR /root/

ENV VERSION 2.28
RUN curl -SL https://www.kernel.org/pub/linux/utils/util-linux/v$VERSION/util-linux-$VERSION.tar.gz | tar -zxf- \
    && ln -s util-linux-$VERSION util-linux \
    && cd /root/util-linux \
    && ./configure --without-ncurses

WORKDIR /root/util-linux
COPY nsenter.c ./sys-utils/nsenter.c
RUN make LDFLAGS=-all-static nsenter \
    && mv nsenter /
