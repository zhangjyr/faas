REGISTRY ?= zhangjyr
IMAGE = $(REGISTRY)/chelloworld

TAG = 0.2

CFLAGS = -Os -Wall -Werror -static # -DVERSION=v$(TAG)-$(REV)

all: container push

container:
	docker build --pull -t $(IMAGE):$(TAG) .

push:
	docker push $(IMAGE):$(TAG)
