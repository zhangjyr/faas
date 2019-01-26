REGISTRY ?= zhangjyr
IMAGE = $(REGISTRY)/nodejsenv

TAG = b1

all: container push

container:
	docker build -t $(IMAGE):$(TAG) --build-arg NODE_ENV=production .

push:
	docker push $(IMAGE):$(TAG)

validate:
	npm install
	mocha --ui=exports
	rm -rf node_modules
	rm -f package-lock.json
