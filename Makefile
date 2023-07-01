APP = reload

GIT_BRANCH = $(shell git rev-parse --abbrev-ref HEAD)
GIT_COMMIT = $(shell git rev-parse --short HEAD)
GIT_TAG = $(shell git tag --points-at HEAD)

.PHONY: deps
deps:
	go mod tidy && go mod vendor

.PHONY: test
test:
	go test ./...

.PHONY: lint
lint:
	go fmt ./...
	go vet ./...

.PHONY: build
build:
	go build -mod vendor \
		-o out/$(APP) \
		.

.PHONY: install
install: build
	mv out/$(APP) ${GOPATH}/bin/$(APP)

.PHONY: build/linux-amd64
build/linux-amd64:
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux \
	go build -mod vendor \
		-o out/$(APP)-linux-amd64 \
		.
