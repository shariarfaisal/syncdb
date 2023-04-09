test: 
	go test -v -cover ./...

server:
	go run main.go

install:
	go mod tidy

.PHONY:
	test server