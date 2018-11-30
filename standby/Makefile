# Copyright 2016 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

REGISTRY ?= zhangjyr
IMAGE = $(REGISTRY)/standby

TAG = b3
# REV = $(shell git describe --contains --always --match='v*')

CFLAGS = -Os -Wall -Werror -static # -DVERSION=v$(TAG)-$(REV)

BIN = standby
SRCS = standby.c

all: container push

$(BIN): $(SRCS)
	mkdir -p bin
	gcc $(CFLAGS) -o bin/$@ $^

container:
	docker build --pull -t $(IMAGE):$(TAG) .
#	touch $@

push:
	docker push $(IMAGE):$(TAG)
#	touch $@

clean:
	rm -rf bin/
