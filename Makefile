BIN := lingo

.PHONY: build run test clean install

build:
	go build -o $(BIN) .

run: build
	./$(BIN) web

test:
	go test ./...

clean:
	rm -f $(BIN)

install: build
	cp $(BIN) /usr/local/bin/$(BIN)
