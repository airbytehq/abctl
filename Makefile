GOPATH := $(shell go env GOPATH)
ABCTL_VERSION?=dev

.PHONY: build
build:
	CGO_ENABLED=0 go build -trimpath -o build/ -ldflags "-w -X github.com/airbytehq/abctl/internal/build.Version=$(ABCTL_VERSION)" .

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
	mockgen --source internal/ui/ui.go -destination internal/ui/mock/mock.go -package mock
	mockgen --source internal/airbox/config_store.go -destination internal/airbox/mock/config.go -package mock
	mockgen --source internal/auth/auth.go -destination internal/auth/mocks_creds_test.go -package auth
	mockgen --source internal/api/client.go -destination internal/api/mock/mock.go -package mock
	mockgen --source internal/k8s/cluster.go -destination internal/k8s/mock/cluster.go -package mock

.PHONY: tools
tools:
	go install go.uber.org/mock/mockgen@$(shell go list -m -f '{{.Version}}' go.uber.org/mock)
