DOCKER_APP_CONTAINER_NAME = externalissuesource
DOCKER_RUN = docker-compose run --service-ports ${DOCKER_APP_CONTAINER_NAME}
GOFILES_EXCLUDING_VENDOR = $(shell find . -type f -name '*.go' -not -path './vendor/*')

.PHONY: up
up:
	docker-compose up -d --build

.PHONY: dep-init
dep-init:
	${DOCKER_RUN} dep init

.PHONY: install-deps
install-deps:
	${DOCKER_RUN} dep ensure -v

test:
	${DOCKER_RUN} go test -v github.com/aimeelaplant/externalissuesource github.com/aimeelaplant/externalissuesource/internal/dateutil github.com/aimeelaplant/externalissuesource/internal/stringutil

format:
	${DOCKER_RUN} go fmt ./
