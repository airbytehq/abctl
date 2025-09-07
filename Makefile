GOPATH := $(shell go env GOPATH)
ABCTL_VERSION?=dev

.PHONY: build
build:
	CGO_ENABLED=0 go build -trimpath -o build/ -ldflags "-w -X github.com/airbytehq/abctl/internal/build.Version=$(ABCTL_VERSION)" .

.PHONY: build-new
build-new:
	CGO_ENABLED=0 go build -tags newcli -trimpath -o build/abctl-new -ldflags "-w -X github.com/airbytehq/abctl/internal/build.Version=$(ABCTL_VERSION)" main_new.go

.PHONY: clean
clean:
	rm -rf build/

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: test
test:
	go test ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: mocks
mocks:
	mockgen --source $(GOPATH)/pkg/mod/github.com/mittwald/go-helm-client@v0.12.15/interface.go -destination internal/helm/mock/mock.go -package mock
	mockgen --source internal/http/client.go -destination internal/http/mock/mock.go -package mock

.PHONY: tools
tools:
	go install go.uber.org/mock/mockgen@$(shell go list -m -f '{{.Version}}' go.uber.org/mock)
