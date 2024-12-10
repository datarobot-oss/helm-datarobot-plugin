APP_NAME := helm-datarobot
SRC_DIR := .
GO := go

# Commands
all: build docs test

build: clean fmt
	@echo "Building the binary..."
	$(GO) build -o $(APP_NAME) $(SRC_DIR)

docs: build
	@echo "Building docs"
	@rm -rf ./docs/
	@mkdir ./docs
	@./$(APP_NAME) docs --path ./docs

pre-test:
	@echo "Pre tests..."
	@helm dependency update testdata/test-chart3
	@helm dependency update testdata/test-chart2
	@helm dependency update testdata/test-chart1

test: pre-test
	@echo "Running tests..."
	$(GO) test ./... -v

clean:
	@echo "Cleaning up..."
	@rm -rf $(APP_NAME)

fmt:
	@echo "Formatting the code..."
	$(GO) fmt ./...

vet:
	@echo "Vet the code..."
	$(GO) vet ./...

lint:
	@echo "Linting the code..."
	@golint ./...
