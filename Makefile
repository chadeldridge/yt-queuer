# Makefile

ROOT_DIR:=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
TARGET_DIR=${ROOT_DIR}/bin
TARGET_NAME=ytqueuer
TARGET_APP=$(TARGET_DIR)/$(TARGET_NAME)
PACKAGE_DIR=${ROOT_DIR}/pkg
GOOS=linux

tidy:
	@go mod tidy
.PHONY: tidy

build: go-build package
.PHONY: build

build-arm64: go-build-arm64 package
.PHONY: build-arm64

run: build
	@${TARGET_DIR}/${TARGET_NAME}
.PHONY: run

go-build:
	@if [ ! -d ${TARGET_DIR} ]; then mkdir ${TARGET_DIR}; fi && cd cmd/server/ && go build -o ${TARGET_APP}
.PHONY: go-build

go-build-arm64:
	@GOARCH=arm64
	@TARGET_NAME=${TARGET_NAME}_arm64
	@TARGET_APP=${TARGET_DIR}/${TARGET_NAME}
	@if [ ! -d ${TARGET_DIR} ]; then mkdir ${TARGET_DIR}; fi && cd cmd/server/ && go build -o ${TARGET_APP}
.PHONY: go-build-arm64

package:
	@if [ ! -d ${PACKAGE_DIR} ]; then mkdir ${PACKAGE_DIR}; fi && cp ${TARGET_APP} ${PACKAGE_DIR}/ && cp LICENSE ${PACKAGE_DIR} && cp -r public ${PACKAGE_DIR}/
.PHONY: package
