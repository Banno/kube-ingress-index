VERSION := $(shell grep -Eo '([0-9]+[\.][0-9]+[\.][0-9]+(-dev)?)' main.go | head -n1)

.PHONY: build docker

build:
	go fmt ./...
	go vet ./...
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -installsuffix cgo -ldflags="-s -w"
	upx --brute kube-ingress-index

docker: build
	docker build -t banno/kube-ingress-index:$(VERSION) -f Dockerfile .
