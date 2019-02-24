REGISTRY ?= zhangjyr
IMAGE = $(REGISTRY)/go2env

TAG = b1

all: container push

container:
	docker build -t $(IMAGE):$(TAG) .

push:
	docker push $(IMAGE):$(TAG)
