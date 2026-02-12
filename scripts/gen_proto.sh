#!/bin/bash

# 安装 protoc 插件
# go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
# go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

PROTO_DIR="api/proto"
OUT_DIR="api/proto"

# 生成 Go 代码
protoc \
  --proto_path=${PROTO_DIR} \
  --go_out=${OUT_DIR} \
  --go_opt=paths=source_relative \
  --go-grpc_out=${OUT_DIR} \
  --go-grpc_opt=paths=source_relative \
  ${PROTO_DIR}/wallet/v1/*.proto

echo "Proto files generated successfully!"

