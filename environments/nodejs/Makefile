REGISTRY ?= zhangjyr
IMAGE = $(REGISTRY)/nodejsenv

TAG = b1

all: container push

container:
	docker build -t $(IMAGE):$(TAG) .

push:
	docker push $(IMAGE):$(TAG)
