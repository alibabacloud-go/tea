all:

fmt:
	go fmt ./

test:
	go test -race -coverprofile=coverage.txt -covermode=atomic ./tea
	go tool cover -html=coverage.txt -o coverage.html
