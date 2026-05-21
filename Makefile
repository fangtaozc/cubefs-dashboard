IMAGE ?= hub.shiyak-office.com/storage/cubefs-dashboard
TAG ?= v1.0.4

.PHONY: image-build image-push

image:
	./deploy/build-image.sh $(IMAGE) $(TAG)
	docker push $(IMAGE):$(TAG)

image-push:
	docker push $(IMAGE):$(TAG)
