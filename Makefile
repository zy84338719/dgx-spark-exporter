BINARY_NAME=dgx-spark-exporter
VERSION?=1.0.0
REVISION?=$(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BRANCH?=$(shell git symbolic-ref --short HEAD 2>/dev/null || echo main)
BUILD_TIME=$(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
CMD_PATH=./cmd/dgx-spark-exporter
LDFLAGS=-ldflags "-s -w -X main.Version=$(VERSION) -X main.Revision=$(REVISION) -X main.Branch=$(BRANCH) -X main.BuildTime=$(BUILD_TIME)"
PREFIX?=/usr/local/bin

.PHONY: all build clean test install fmt vet lint run
.PHONY: service-install service-uninstall service-start service-stop service-restart service-status

all: fmt vet build

build:
	go build $(LDFLAGS) -o $(BINARY_NAME) $(CMD_PATH)

install:
	go install $(LDFLAGS) $(CMD_PATH)

clean:
	go clean
	rm -f $(BINARY_NAME)

test:
	go test -v -race ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

lint:
	golangci-lint run ./...

run:
	go run $(LDFLAGS) $(CMD_PATH)

docker-build:
	docker build -t $(BINARY_NAME):$(VERSION) .

service-install: build
	@echo "[1/4] Stopping existing service if running..."
	-systemctl stop $(BINARY_NAME) 2>/dev/null || true
	@echo "[2/4] Installing binary to $(PREFIX)/..."
	install -m 755 $(BINARY_NAME) $(PREFIX)/$(BINARY_NAME)
	@echo "[3/4] Installing systemd service..."
	install -m 644 deploy/$(BINARY_NAME).service /etc/systemd/system/$(BINARY_NAME).service
	systemctl daemon-reload
	@echo "[4/4] Enabling and starting service..."
	systemctl enable $(BINARY_NAME)
	systemctl start $(BINARY_NAME)
	@echo ""
	@echo "Service installed successfully!"
	@echo "Check status: make service-status"
	@echo "View logs:   journalctl -u $(BINARY_NAME) -f"

service-uninstall:
	@echo "[1/3] Stopping service..."
	systemctl stop $(BINARY_NAME) 2>/dev/null || true
	@echo "[2/3] Disabling service..."
	systemctl disable $(BINARY_NAME) 2>/dev/null || true
	@echo "[3/3] Removing files..."
	rm -f /etc/systemd/system/$(BINARY_NAME).service
	rm -f $(PREFIX)/$(BINARY_NAME)
	systemctl daemon-reload
	@echo ""
	@echo "Service uninstalled successfully!"

service-start:
	systemctl start $(BINARY_NAME)
	@echo "Service started."

service-stop:
	systemctl stop $(BINARY_NAME)
	@echo "Service stopped."

service-restart:
	systemctl restart $(BINARY_NAME)
	@echo "Service restarted."

service-status:
	@systemctl status $(BINARY_NAME) --no-pager || true

help:
	@echo "DGX Spark Exporter - Makefile Commands"
	@echo ""
	@echo "Build:"
	@echo "  make build          - Build the binary"
	@echo "  make clean          - Remove binary"
	@echo "  make install        - Install to GOPATH/bin"
	@echo "  make docker-build   - Build Docker image"
	@echo ""
	@echo "Development:"
	@echo "  make run            - Run the application"
	@echo "  make test           - Run tests"
	@echo "  make fmt            - Format code"
	@echo "  make vet            - Run go vet"
	@echo "  make lint           - Run golangci-lint"
	@echo ""
	@echo "Systemd Service:"
	@echo "  make service-install   - Install and start systemd service (needs sudo)"
	@echo "  make service-uninstall - Remove systemd service (needs sudo)"
	@echo "  make service-start     - Start service"
	@echo "  make service-stop      - Stop service"
	@echo "  make service-restart   - Restart service"
	@echo "  make service-status    - Show service status"
