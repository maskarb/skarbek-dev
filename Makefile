

IMAGE_TAG_BASE ?= localhost:32000/skarbek-dev
GIT_COMMIT ?= $(shell git rev-parse HEAD)
IMG ?= $(IMAGE_TAG_BASE):$(GIT_COMMIT)

docker-build:
	docker build -t ${IMG} .

docker-push:
	docker push ${IMG}

