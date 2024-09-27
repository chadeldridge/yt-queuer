# Makefile

ROOT_DIR:=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
TARGET_DIR=${ROOT_DIR}/bin
APP_NAME=ytqueuer
TARGET_APP=$(TARGET_DIR)/$(APP_NAME)
PACKAGE_DIR=${ROOT_DIR}/pkg
GOOS=linux
GOARCH=amd64

tidy:
	@go mod tidy
.PHONY: tidy

build: npm-build go-build package
	@echo "Done.\n"
.PHONY: build

build-arm64: npm-build go-build-arm64 go-build package
	@echo "Done.\n"
.PHONY: build-arm64

run: npm-build build
	@${TARGET_DIR}/${APP_NAME}
.PHONY: run

npm-build:
	@echo "Building frontend..."
	@npm run build
.PHONY: npm-build

go-build:
	@echo "Building for ${GOOS}/${GOARCH}..."
	@if [ ! -d ${TARGET_DIR} ]; then mkdir ${TARGET_DIR}; fi && cd cmd/server/ && env GOOS=${GOOS} GOARCH=${GOARCH} go build -o ${TARGET_APP} && cp ${TARGET_APP} ${PACKAGE_DIR}/ytqueuer
.PHONY: go-build

go-build-arm64:
	$(eval GOARCH=arm64)
	$(eval TARGET_NAME=${APP_NAME}_${GOARCH})
	$(eval TARGET_APP=${TARGET_DIR}/${TARGET_NAME})
.PHONY: go-build-arm64

package:
	@echo "Packaging..."
	@if [ ! -d ${PACKAGE_DIR} ]; then mkdir ${PACKAGE_DIR}; fi || exit 1
	@cp ${TARGET_APP} ${PACKAGE_DIR}/${APP_NAME}
	@cp LICENSE ${PACKAGE_DIR}/
	@cp -r public ${PACKAGE_DIR}/
.PHONY: package
