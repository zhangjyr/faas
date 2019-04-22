REGISTRY ?= zhangjyr
IMAGE = $(REGISTRY)/goenv

TAG = b1

all: container push

container:
	docker build --pull -t $(IMAGE):$(TAG) .

push:
	docker push $(IMAGE):$(TAG)
