# Makefile — các lệnh hay dùng. Trên Windows chạy qua Git Bash, hoặc dùng lệnh go trực tiếp.
# DATABASE_URL đọc từ môi trường (hoặc file .env khi chạy `go run`).

.PHONY: run build test test-int vet migrate-up migrate-down migrate-new seed

run:          ## chạy server
	go run ./cmd/api

build:        ## build binary vào bin/api
	go build -o bin/api ./cmd/api

test:         ## chạy unit test
	go test ./...

test-int:     ## chạy integration test (chạm DB thật)
	go test -tags=integration ./test/...

vet:          ## kiểm tra tĩnh
	go vet ./...

migrate-up:   ## áp toàn bộ migration
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down: ## rollback 1 bước migration
	migrate -path migrations -database "$(DATABASE_URL)" down 1

migrate-new:  ## tạo migration mới:  make migrate-new name=create_posts
	migrate create -ext sql -dir migrations -seq $(name)

seed:         ## nạp dữ liệu mẫu
	psql "$(DATABASE_URL)" -f scripts/seed.sql
