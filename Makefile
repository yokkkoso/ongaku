-include ./config.make

BINARY := bot
WORK_DIR := $(abspath .)
BINARY_PATH  := $(abspath ./$(BINARY))
SERVICE_DEST := /etc/systemd/system/$(SERVICE_NAME)
TEMPLATE     := template.service
CURRENT_USER := $(shell whoami)

export CGO_ENABLED := 1

.PHONY: build-info
build-info:
ifndef GITHASH
	$(eval GITHASH := $(shell git rev-parse --short HEAD))
endif
ifndef BUILD_DATE
	$(eval BUILD_DATE := $(shell date +"%Y-%m-%d %H:%M:%S"))
endif

.PHONY: build-flags
build-flags: build-info
	$(eval BUILD_LDFLAGS := $(LDFLAGS))
	$(eval BUILD_LDFLAGS += -X 'github.com/yokkkoso/musicbot/internal/build.githash=$(GITHASH)')
	$(eval BUILD_LDFLAGS += -X 'github.com/yokkkoso/musicbot/internal/build.buildstamp=$(BUILD_DATE)')
	$(eval BUILD_FLAGS := -ldflags "$(BUILD_LDFLAGS)" -o $(BINARY))

.PHONY: build
build: build-flags
	go build $(BUILD_FLAGS) ./cmd/music/main.go

service-install: build
	@echo "Установка systemd-сервиса для $(BINARY)"
	@echo "   Бинарник: $(BINARY_PATH)"
	@echo "   Цель    : $(SERVICE_DEST)"

	@sed \
		-e "s|ExecStart=CHANGE_ME|ExecStart=$(BINARY_PATH)|" \
		-e "s|WorkingDirectory=CHANGE_ME|WorkingDirectory=$(WORK_DIR)|" \
		-e "s|^User=CHANGE_ME|User=$(CURRENT_USER)|" \
		$(TEMPLATE) | \
		sudo tee $(SERVICE_DEST) > /dev/null

	@sudo chmod 644 $(SERVICE_DEST)

	@echo "Перезагрузка конфигурации systemd..."
	@sudo systemctl daemon-reload

	@echo ""
	@echo "Сервис $(SERVICE_NAME) установлен"
	@echo ""

.PHONY: service-enable service-start service-restart service-status
service-enable:
	@sudo systemctl enable $(SERVICE_NAME)

service-start:
	@sudo systemctl start $(SERVICE_NAME)

service-restart:
	@sudo systemctl restart $(SERVICE_NAME)

service-status:
	@sudo systemctl status $(SERVICE_NAME) --no-pager -l --lines=20

.PHONY: fmt
fmt:
	go fmt ./...

migration-sql:
	@if [ -z "${name}" ]; then echo "Usage: make migration-sql name=name_of_migration_file"; exit 1; fi
	go run github.com/pressly/goose/v3/cmd/goose@latest -dir internal/database/migrations create ${name} sql
.PHONY: migration

migration-go:
	@if [ -z "${name}" ]; then echo "Usage: make migration-go name=name_of_migration_file"; exit 1; fi
	go run github.com/pressly/goose/v3/cmd/goose@latest -dir internal/database/migrations create ${name}
.PHONY: migration

.PHONY: lint
lint:
	golangci-lint run

.PHONY: run
run: build
	./$(BINARY)
