REGISTRY ?= zhangjyr
IMAGE = $(REGISTRY)/sleep

TAG = 0.2

CFLAGS = -Os -Wall -Werror -static

BIN = sleep
SRCS = sleep.c

all: container push

$(BIN): $(SRCS)
	mkdir -p bin
	gcc $(CFLAGS) -o bin/$@ $^

container:
	docker build --pull -t $(IMAGE):$(TAG) .

push:
	docker push $(IMAGE):$(TAG)

clean:
	rm -rf bin/
