.PHONY: build build-pi run clean

build:
	go build -o bin/economizador ./cmd/economizador

build-pi:
	GOOS=linux GOARCH=arm64 go build -o bin/economizador-pi ./cmd/economizador

run: build
	./bin/economizador

clean:
	rm -rf bin/
