.PHONY: proto-go proto-python

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