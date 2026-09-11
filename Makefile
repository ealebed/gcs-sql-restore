VERSION=$(shell date -u '+%y.%m.%d-%H.%M')

GO            := go
GO111MODULE   := on
CGO_ENABLED   := 0
GOBUILD       := CGO_ENABLED=$(CGO_ENABLED) GO111MODULE=$(GO111MODULE) $(GO) build
GOTEST        := $(GO) test -gcflags='-l' -p 3
GOLANGCI_LINT := golangci-lint
BIN           := bin/gcs-sql-restore

.PHONY: all
all: build

.PHONY: build
build:
	$(GO) build ./...

.PHONY: test
test:
	$(GOTEST) ./...

.PHONY: test-race
test-race:
	$(GO) test ./... -race

.PHONY: fmt
fmt:
	$(GO) fmt ./...
	gofmt -s -w .

.PHONY: lint
lint:
	$(GOLANGCI_LINT) run --timeout 4m --config .golangci.yaml

.PHONY: tidy
tidy:
	$(GO) mod tidy
	$(GO) mod verify

.PHONY: terraform-fmt
terraform-fmt:
	terraform -chdir=terraform fmt -recursive

.PHONY: terraform-validate
terraform-validate:
	terraform -chdir=terraform init -backend=false -input=false -lockfile=readonly
	terraform -chdir=terraform validate

.PHONY: terraform-lint
terraform-lint:
	cd terraform && tflint --init && tflint

.PHONY: clean
clean:
	rm -f $(BIN)
	rm -rf bin/
	rm -f coverage.out
	rm -f *.test

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  all                - build the project (default)"
	@echo "  build              - build the project"
	@echo "  test               - run tests"
	@echo "  test-race          - run tests with race detector"
	@echo "  fmt                - format Go code"
	@echo "  lint               - run golangci-lint"
	@echo "  tidy               - tidy and verify modules"
	@echo "  terraform-fmt      - format Terraform files"
	@echo "  terraform-validate - terraform init -backend=false and validate"
	@echo "  terraform-lint     - tflint on terraform/"
	@echo "  clean              - clean build artifacts"
	@echo "  help               - show this help message"
