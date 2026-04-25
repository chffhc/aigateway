IMAGE ?= aigateway
TAG ?= latest

.PHONY: build docker-build docker-run docker-buildx bootstrap-buildx

build:
	go build ./cmd/server

docker-build:
	DOCKER_BUILDKIT=1 docker build -t $(IMAGE):$(TAG) .

bootstrap-buildx:
	docker buildx inspect multiarch >/dev/null 2>&1 || docker buildx create --name multiarch --use
	docker buildx inspect --bootstrap

docker-buildx: bootstrap-buildx
	docker buildx bake --set *.args.GO_VERSION=1.22 --set *.args.ALPINE_VERSION=3.19 \
		--set local.tags=$(IMAGE):$(TAG) local

release: bootstrap-buildx
	docker buildx bake --push --set release.tags=$(IMAGE):$(TAG) release
