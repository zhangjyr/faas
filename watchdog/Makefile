REGISTRY ?= zhangjyr
IMAGE = $(REGISTRY)/hyperfaas-ics

TAG = b1

all: container push

linux:
	CGO_ENABLED=0 GOOS=linux go build -a -ldflags "-s -w" -installsuffix cgo -o fwatchdog

container:
	docker build --pull -t $(IMAGE):$(TAG) .

push:
	docker push $(IMAGE):$(TAG)
