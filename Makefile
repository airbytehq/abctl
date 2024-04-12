ABCTL_VERSION?=dev
.PHONY: build
build:
	CGO_ENABLED=0 go build -trimpath -o build/ -ldflags "-w -X airbyte.io/abctl/internal/build.Version=$(ABCTL_VERSION)" .

.PHONY: clean
clean:
	rm -rf build/

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: vet
vet:
	go vet ./...