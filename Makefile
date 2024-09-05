all:

fmt:
	go fmt ./tea ./dara ./utils

test:
	go test -race -coverprofile=coverage.txt -covermode=atomic ./tea ./utils ./dara
	go tool cover -html=coverage.txt -o coverage.html
