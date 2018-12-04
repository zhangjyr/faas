REGISTRY ?= zhangjyr
IMAGE = $(REGISTRY)/chelloworld

TAG = b5

CFLAGS = -Os -Wall -Werror -static # -DVERSION=v$(TAG)-$(REV)

all: container push

container:
	docker build --pull -t $(IMAGE):$(TAG) .

push:
	docker push $(IMAGE):$(TAG)
