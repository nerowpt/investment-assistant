# investment-assistant 构建与开发命令
# 见 docs/04-技术架构.md §二十三

.PHONY: help proto proto-go proto-py migrate-up migrate-down test build inv worker-deps

ACCOUNT ?= default
DATA_ROOT ?= ./data
MIGRATE_DB = $(DATA_ROOT)/accounts/$(ACCOUNT)/db/assistant.sqlite
PROTOC_GEN_GO := $(shell go env GOPATH)/bin/protoc-gen-go
PROTOC_GEN_GO_GRPC := $(shell go env GOPATH)/bin/protoc-gen-go-grpc

help:
	@echo "make proto        - 生成 Go + Python gRPC stub"
	@echo "make worker-deps  - 安装 Python data-worker 依赖"
	@echo "make migrate-up   - 对 ACCOUNT 执行 SQLite 迁移"
	@echo "make build        - 编译 cmd/inv"
	@echo "make test         - go test ./..."

proto: proto-go proto-py

proto-go:
	@test -f "$(PROTOC_GEN_GO)" || go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@test -f "$(PROTOC_GEN_GO_GRPC)" || go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	protoc -I proto \
		--go_out=gen/go --go_opt=paths=source_relative \
		--go-grpc_out=gen/go --go-grpc_opt=paths=source_relative \
		proto/common/v1/provenance.proto \
		proto/dataworker/v1/dataworker.proto \
		proto/coreingest/v1/coreingest.proto

proto-py:
	cd services/data-worker && python -m pip install grpcio-tools -q
	cd services/data-worker && python -m grpc_tools.protoc -I ../../proto \
		--python_out=data_worker/pb --grpc_python_out=data_worker/pb \
		../../proto/common/v1/provenance.proto \
		../../proto/dataworker/v1/dataworker.proto

worker-deps:
	cd services/data-worker && python -m pip install -e .

migrate-up:
	@mkdir -p $(dir $(MIGRATE_DB))
	migrate -path migrations -database "sqlite3://$(MIGRATE_DB)" up

migrate-down:
	migrate -path migrations -database "sqlite3://$(MIGRATE_DB)" down 1

build:
	go build -o bin/inv.exe ./cmd/inv

test:
	go test ./...

inv: build
	./bin/inv.exe version
