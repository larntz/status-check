BINARY_NAME=status
 
all: build test
 
build:
	go build -o ${BINARY_NAME} ./cmd/main.go
 
test:
	go test -race -coverprofile=cover.p -v ./cmd/worker
	go tool cover -func=cover.p
	rm cover.p
 
run-worker:
	go build -o ${BINARY_NAME} ./cmd/main.go
	./${BINARY_NAME} worker 
 
clean:
	go clean
	rm ${BINARY_NAME}
