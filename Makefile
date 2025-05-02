GO ?= go
GOFMT ?= gofmt "-s"
PACKAGES ?= $(shell $(GO) list ./...)
GOFILES := $(shell find . -name "*.go" -type f -not -path './vendor/*')
COMPOSEFILES_ACCEPTANCE_TESTING = -f tests/docker-compose.yml -f tests/docker-compose.acceptance.yml


.PHONY: all
all: fmt lint vet test

.PHONY: build
build:
	$(GO) build -o integration-test-runner .

.PHONY: test
test:
	$(GO) test -cover -coverprofile=coverage.txt $(PACKAGES) && echo "\n==>\033[32m Ok\033[m\n" || exit 1

.PHONY: test-short
test-short:
	$(GO) test -cover -coverprofile=coverage.txt --short $(PACKAGES) && echo "\n==>\033[32m Ok\033[m\n" || exit 1

.PHONY: fmt
fmt:
	$(GOFMT) -w $(GOFILES)

.PHONY: lint
lint:
	for pkg in ${PACKAGES}; do \
		golint -set_exit_status $$pkg || GOLINT_FAILED=1; \
	done; \
	[ -z "$$GOLINT_FAILED" ]

.PHONY: vet
vet:
	$(GO) vet $(PACKAGES)

.PHONY: clean
clean:
	$(GO) clean -modcache -x -i ./...
	find . -name coverage.txt -delete

.PHONY: acceptance-testing-build
acceptance-testing-build:
	docker compose $(COMPOSEFILES_ACCEPTANCE_TESTING) build

.PHONY: acceptance-testing-up
acceptance-testing-up:
	docker compose $(COMPOSEFILES_ACCEPTANCE_TESTING) up -d

.PHONY: acceptance-testing-run
acceptance-testing-run:
	docker compose $(COMPOSEFILES_ACCEPTANCE_TESTING) exec acceptance-testing /testing/run.sh

.PHONY: acceptance-testing-update-golden-files
acceptance-testing-update-golden-files:
	docker compose $(COMPOSEFILES_ACCEPTANCE_TESTING) exec acceptance-testing /testing/run.sh --update-goldens

.PHONY: acceptance-testing-logs
acceptance-testing-logs:
	docker compose $(COMPOSEFILES_ACCEPTANCE_TESTING) ps -a
	docker compose $(COMPOSEFILES_ACCEPTANCE_TESTING) logs

.PHONY: acceptance-testing-down
acceptance-testing-down:
	docker compose $(COMPOSEFILES_ACCEPTANCE_TESTING) down
