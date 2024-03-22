build:
	CGO_ENABLED=0 go build -trimpath -o build/ -ldflags "-w" .

clean:
	rm -rf build/

fmt:
	go fmt ./...