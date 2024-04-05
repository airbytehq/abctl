.PHONY:
build:
	CGO_ENABLED=0 go build -trimpath -o build/ -ldflags "-w" .

.PHONY:
clean:
	rm -rf build/

.PHONY:
fmt:
	go fmt ./...

.PHONY:
vet:
	go vet ./...