.PHONY: proto-go proto-python docker-build docker-run

CONFIG ?= ./config/local.yaml

proto-go:
	protoc -I proto proto/generator/generator.proto \
		--go_out=./gen/go/ \
		--go_opt=paths=source_relative \
		--go-grpc_out=./gen/go/ \
		--go-grpc_opt=paths=source_relative

proto-python:
	python3 -m grpc_tools.protoc -I proto proto/generator/generator.proto \
        --python_out=./gen/python \
        --pyi_out=./gen/python \
        --grpc_python_out=./gen/python

docker-build:
	docker build -t agreements-generator-server .

docker-run:
	docker run -p 8080:8080 --rm --name agreements-generator-server -v $(CONFIG):

test:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	go tool cover -func=coverage.out

clean:
	rm -f coverage.out coverage.html
