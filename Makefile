

default: build test serve

serve:
	./carserv serve

build:
	go build ./cmd/carserv

test:
	go test ./pkg/carserv/server -v