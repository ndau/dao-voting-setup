OUTPUT_DIR := bin/project

MAIN_GO := ./
build: build-binary

build-binary:
	go build -o $(OUTPUT_DIR) $(MAIN_GO)

test:
	go test ./... -v

test+coverage:
	go test ./... -v -cover -coverprofile=coverprofile.out
	go tool cover -html=coverprofile.out -o coverage.html

