# investment-assistant 构建与开发命令
# 见 docs/04-技术架构.md §二十三

.PHONY: help proto migrate-up migrate-down test build inv

ACCOUNT ?= default
DATA_ROOT ?= ./data
MIGRATE_DB = $(DATA_ROOT)/accounts/$(ACCOUNT)/db/assistant.sqlite

help:
	@echo "make proto        - buf lint + 生成 Go/Python stub"
	@echo "make migrate-up   - 对 ACCOUNT 执行 SQLite 迁移"
	@echo "make migrate-down - 回滚一步"
	@echo "make build        - 编译 cmd/inv"
	@echo "make test         - go test ./..."

proto:
	buf lint proto
	@echo "Go: buf generate proto (需安装 buf CLI)"
	@echo "Python: 见 services/data-worker README"

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
