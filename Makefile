test: 
	go test -v -cover ./...

server:
	go run main.go

.PHONY:
	test server