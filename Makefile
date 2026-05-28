.PHONY: help build run test test-cover vet tidy clean

# --- переменные ---
BINARY_NAME ?= gophermart
BINARY_PATH ?= cmd/gophermart/$(BINARY_NAME)
MAIN_PACKAGE ?= ./cmd/gophermart
GO ?= go
GOFLAGS ?= -buildvcs=false

RUN_ADDRESS ?= localhost:8080
DATABASE_URI ?= postgresql://postgres:postgres@localhost:5432/praktikum?sslmode=disable
ACCRUAL_SYSTEM_ADDRESS ?=


# --- сборка и запуск ---
build:
	cd cmd/gophermart && $(GO) build $(GOFLAGS) -o $(BINARY_NAME) .

run: build
	cd cmd/gophermart && \
		RUN_ADDRESS=$(RUN_ADDRESS) \
		DATABASE_URI=$(DATABASE_URI) \
		ACCRUAL_SYSTEM_ADDRESS=$(ACCRUAL_SYSTEM_ADDRESS) \
		./$(BINARY_NAME) -a $(RUN_ADDRESS) -d "$(DATABASE_URI)" -r "$(ACCRUAL_SYSTEM_ADDRESS)"

# --- качество кода ---
test:
	$(GO) test ./... -count=1

test-cover:
	$(GO) test ./... -count=1 -coverprofile=coverage.out
	$(GO) tool cover -func=coverage.out

vet:
	$(GO) vet ./...

tidy:
	$(GO) mod tidy

# --- очистка ---
clean:
	rm -f cmd/gophermart/gophermart cmd/gophermart/gophermart.exe coverage.out
