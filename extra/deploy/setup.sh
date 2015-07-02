#! /bin/bash

# Install golang protobuf compiler
go get -u github.com/golang/protobuf/{proto,protoc-gen-go}

# Install dependencies
echo `go get -u ./...`