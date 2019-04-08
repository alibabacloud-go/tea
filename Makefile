all:

fmt:
	go fmt ./

test:
	go test -race -coverprofile=coverage.txt -covermode=atomic ./
	go tool cover -html=coverage.txt -o coverage.html
